package services

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"plumelauncher/internal/auth"
	"plumelauncher/internal/skins"
)

// appearanceFixture sets up a Microsoft account with a valid token plus a
// fake Mojang server, so appearance operations have something to talk to.
func appearanceFixture(t *testing.T) (*fakeMojang, *SkinService) {
	t.Helper()
	f, svc, _ := startFakeMojang(t)
	keys := msMemoryKeyring{}
	svc.Keyring = keys
	svcAccount := svc.accountService()
	if _, err := svcAccount.upsertMicrosoftAccount(&auth.MicrosoftCredentials{
		AccessToken:  "test-token",
		RefreshToken: "refresh-token",
		ExpiresAt:    time.Now().Add(time.Hour),
		UUID:         "uuid-1",
		Username:     "Steve",
	}); err != nil {
		t.Fatalf("upsertMicrosoftAccount: %v", err)
	}
	svc.Debouncer.Reset()
	return f, svc
}

// testPNG builds a valid modern skin PNG for fixtures.
func appearanceTestPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 120, G: 80, B: 40, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// fakeMojang records requests and answers with a canned profile.
type fakeMojang struct {
	server          *httptest.Server
	uploadCount     int
	fetchCount      int
	resetCount      int
	equipCapeCount  int
	hideCapeCount   int
	uploadVariant   string
	uploadContentType string
}

func (f *fakeMojang) profileJSON() []byte {
	profile := skins.Profile{
		ID:   "uuid-1",
		Name: "Steve",
		Skins: []skins.ProfileSkin{{
			ID: "skin-1", State: "ACTIVE", URL: "https://textures.minecraft.net/texture/abc", Variant: "CLASSIC",
		}},
		Capes: []skins.ProfileCape{{
			ID: "cape-1", State: "INACTIVE", URL: "https://textures.minecraft.net/texture/cape", Alias: "Minecon",
		}},
	}
	payload, _ := json.Marshal(profile)
	return payload
}

func startFakeMojang(t *testing.T) (*fakeMojang, *SkinService, string) {
	t.Helper()
	f := &fakeMojang{}
	mux := http.NewServeMux()
	mux.HandleFunc("/minecraft/profile", func(w http.ResponseWriter, r *http.Request) {
		f.fetchCount++
		if r.Header.Get("Authorization") != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
			return
		}
		_, _ = w.Write(f.profileJSON())
	})
	mux.HandleFunc("/minecraft/profile/skins", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			f.resetCount++
			_, _ = w.Write(f.profileJSON())
			return
		}
		f.uploadCount++
		f.uploadVariant = r.FormValue("variant")
		f.uploadContentType = r.Header.Get("Content-Type")
		if r.Header.Get("Authorization") != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if err := r.ParseMultipartForm(2 << 20); err != nil {
			w.WriteHeader(http.StatusUnsupportedMediaType)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "UnsupportedMediaType"})
			return
		}
		_, _ = w.Write(f.profileJSON())
	})
	mux.HandleFunc("/minecraft/profile/skins/active", func(w http.ResponseWriter, r *http.Request) {
		f.resetCount++
		_, _ = w.Write(f.profileJSON())
	})
	mux.HandleFunc("/minecraft/profile/capes/active", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			f.equipCapeCount++
		case http.MethodDelete:
			f.hideCapeCount++
		}
		_, _ = w.Write(f.profileJSON())
	})

	f.server = httptest.NewServer(mux)
	t.Cleanup(f.server.Close)

	svc := NewSkinService(t.TempDir())
	svc.Mojang.BaseURL = f.server.URL
	svc.Mojang.HTTPClient = f.server.Client()
	// The debouncer window is cleared per call in tests that need it.
	svc.Debouncer.Reset()
	return f, svc, f.server.URL
}

func TestUploadSkinHappyPath(t *testing.T) {
	f, svc := appearanceFixture(t)

	pngData := appearanceTestPNG(t)
	result, err := svc.UploadSkin("uuid-1", pngData, skins.VariantSlim)
	if err != nil {
		t.Fatalf("UploadSkin: %v", err)
	}
	if f.uploadCount != 1 {
		t.Fatalf("upload count = %d, want 1", f.uploadCount)
	}
	if f.uploadVariant != "slim" {
		t.Errorf("variant = %q, want slim", f.uploadVariant)
	}
	if result.Profile == nil || result.Profile.ID != "uuid-1" {
		t.Errorf("profile = %+v, want uuid-1", result.Profile)
	}
	if len(result.LocalSkins) != 1 {
		t.Errorf("local skins = %d, want 1 (closet save)", len(result.LocalSkins))
	}
}

func TestUploadSkinNormalizesLegacyInput(t *testing.T) {
	f, svc := appearanceFixture(t)

	// Build a legacy 64x32 PNG.
	img := image.NewNRGBA(image.Rect(0, 0, 64, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 64; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 9, G: 9, B: 9, A: 255})
		}
	}
	var legacy bytes.Buffer
	if err := png.Encode(&legacy, img); err != nil {
		t.Fatal(err)
	}

	if _, err := svc.UploadSkin("uuid-1", legacy.Bytes(), skins.VariantClassic); err != nil {
		t.Fatalf("UploadSkin legacy: %v", err)
	}
	if f.uploadCount != 1 {
		t.Fatalf("upload count = %d", f.uploadCount)
	}
}

func TestUploadSkinRejectsInvalidPNG(t *testing.T) {
	f, svc := appearanceFixture(t)
	_, err := svc.UploadSkin("uuid-1", []byte("not a png"), skins.VariantClassic)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if f.uploadCount != 0 {
		t.Errorf("upload count = %d, want 0 (rejected locally)", f.uploadCount)
	}
}

func TestUploadSkinDebouncesRapidCalls(t *testing.T) {
	f, svc := appearanceFixture(t)
	pngData := appearanceTestPNG(t)

	if _, err := svc.UploadSkin("uuid-1", pngData, skins.VariantClassic); err != nil {
		t.Fatalf("first upload: %v", err)
	}
	_, err := svc.UploadSkin("uuid-1", pngData, skins.VariantClassic)
	if err == nil {
		t.Fatal("second immediate upload should be debounced")
	}
	if f.uploadCount != 1 {
		t.Errorf("upload count = %d, want 1 (debounced)", f.uploadCount)
	}
}

func TestUploadSkinTokenExpiredSurfacesUpstream(t *testing.T) {
	_, svc := appearanceFixture(t)
	svc.Mojang.BaseURL = "http://127.0.0.1:1" // unreachable: upstream failure path
	svc.Debouncer.Reset()
	_, err := svc.UploadSkin("uuid-1", appearanceTestPNG(t), skins.VariantClassic)
	if err == nil {
		t.Fatal("expected an upstream error for an unreachable Mojang")
	}
}

func TestResetSkinAndCapes(t *testing.T) {
	f, svc := appearanceFixture(t)

	if _, err := svc.ResetSkin("uuid-1"); err != nil {
		t.Fatalf("ResetSkin: %v", err)
	}
	if f.resetCount == 0 {
		t.Error("reset not called")
	}

	svc.Debouncer.Reset()
	if _, err := svc.EquipCape("uuid-1", "cape-1"); err != nil {
		t.Fatalf("EquipCape: %v", err)
	}
	if f.equipCapeCount != 1 {
		t.Errorf("equip cape count = %d", f.equipCapeCount)
	}

	svc.Debouncer.Reset()
	if _, err := svc.HideCape("uuid-1"); err != nil {
		t.Fatalf("HideCape: %v", err)
	}
	if f.hideCapeCount != 1 {
		t.Errorf("hide cape count = %d", f.hideCapeCount)
	}
}

func TestSkinDebouncerWindow(t *testing.T) {
	d := NewSkinDebouncer(50 * time.Millisecond)
	if !d.Acquire() {
		t.Fatal("first Acquire = false")
	}
	if d.Acquire() {
		t.Error("second immediate Acquire = true, want false")
	}
	time.Sleep(60 * time.Millisecond)
	if !d.Acquire() {
		t.Error("Acquire after window = false, want true")
	}
}

func TestClosetRoundTrip(t *testing.T) {
	svc := NewSkinService(t.TempDir())
	pngData := appearanceTestPNG(t)

	key, err := svc.SaveLocalSkin(pngData, skins.VariantSlim)
	if err != nil {
		t.Fatalf("SaveLocalSkin: %v", err)
	}
	saved, err := svc.LoadLocalSkinPNG(key)
	if err != nil {
		t.Fatalf("LoadLocalSkinPNG: %v", err)
	}
	if !bytes.Equal(saved, pngData) {
		t.Error("closet PNG does not round-trip")
	}

	// Idempotent save: same content, same key, one entry.
	key2, err := svc.SaveLocalSkin(pngData, skins.VariantSlim)
	if err != nil {
		t.Fatalf("SaveLocalSkin again: %v", err)
	}
	if key2 != key {
		t.Errorf("key changed: %q vs %q", key, key2)
	}
	all, err := svc.ListLocalSkins()
	if err != nil {
		t.Fatalf("ListLocalSkins: %v", err)
	}
	if len(all) != 1 {
		t.Errorf("closet entries = %d, want 1", len(all))
	}

	if err := svc.DeleteLocalSkin(key); err != nil {
		t.Fatalf("DeleteLocalSkin: %v", err)
	}
	if _, err := svc.LoadLocalSkinPNG(key); err == nil {
		t.Error("deleted skin still loads")
	}
}
