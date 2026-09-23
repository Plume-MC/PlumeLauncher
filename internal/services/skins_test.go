package services

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var testPNG = []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 's', 'k', 'i', 'n'}

func texturesProperty(t *testing.T, skinURL string) string {
	t.Helper()
	payload, err := json.Marshal(map[string]any{
		"textures": map[string]any{
			"SKIN": map[string]string{"url": skinURL},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(payload)
}

func writeProfile(w http.ResponseWriter, t *testing.T, skinURL string) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"id":         "00000000000000000000000000000000",
		"name":       "Player",
		"properties": []map[string]string{{"name": "textures", "value": texturesProperty(t, skinURL)}},
	})
}

func writePNG(w http.ResponseWriter, r *http.Request) {
	// Public texture endpoints must never require credentials.
	if r.Header.Get("Authorization") != "" || r.Header.Get("X-Session-Token") != "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	_, _ = w.Write(testPNG)
}

// allowTestHost lets the httptest loopback host through the production
// texture allowlist for the duration of one test.
func allowTestHost(t *testing.T) {
	t.Helper()
	skinTextureHosts["127.0.0.1"] = true
	t.Cleanup(func() { delete(skinTextureHosts, "127.0.0.1") })
}

func skinService(dir string, keyring msMemoryKeyring) *AccountService {
	return &AccountService{DataRoot: dir, Keyring: keyring, HTTPClient: http.DefaultClient}
}

func TestGetAccountSkinOfflineUsesDefaultSteve(t *testing.T) {
	allowTestHost(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/SkinTemplates/steve.png" {
			t.Errorf("path = %q", r.URL.Path)
		}
		writePNG(w, r)
	}))
	defer server.Close()
	prev := steveSkinURL
	steveSkinURL = server.URL + "/SkinTemplates/steve.png"
	t.Cleanup(func() { steveSkinURL = prev })

	dir := t.TempDir()
	svc := skinService(dir, msMemoryKeyring{})
	accounts, err := svc.loadAccounts()
	if err != nil {
		t.Fatal(err)
	}
	accounts = append(accounts, Account{UUID: "offline-uuid-1", Username: "Lah", Type: AccountTypeOffline})
	if err := svc.saveAccounts(accounts); err != nil {
		t.Fatal(err)
	}

	dataURL, err := svc.GetAccountSkin("offline-uuid-1")
	if err != nil {
		t.Fatalf("GetAccountSkin: %v", err)
	}
	raw, ok := strings.CutPrefix(dataURL, "data:image/png;base64,")
	if !ok {
		t.Fatalf("dataURL = %q, want data:image/png prefix", dataURL)
	}
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil || string(decoded) != string(testPNG) {
		t.Fatalf("decoded = %q, err = %v", decoded, err)
	}
}

func TestGetAccountSkinElyByReadsSessionProfile(t *testing.T) {
	allowTestHost(t)
	var profileHits, authHeaders int
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/session/profile/"):
			profileHits++
			if r.Header.Get("Authorization") != "" {
				authHeaders++
			}
			if r.URL.Path != "/session/profile/elyuuidnodash" {
				t.Errorf("profile path = %q", r.URL.Path)
			}
			writeProfile(w, t, server.URL+"/skin.png")
		case r.URL.Path == "/skin.png":
			writePNG(w, r)
		default:
			t.Errorf("unexpected path %q", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	prevSession := elySessionProfileURL
	elySessionProfileURL = server.URL + "/session/profile/"
	t.Cleanup(func() { elySessionProfileURL = prevSession })

	dir := t.TempDir()
	svc := skinService(dir, msMemoryKeyring{})
	accounts, err := svc.loadAccounts()
	if err != nil {
		t.Fatal(err)
	}
	accounts = append(accounts, Account{UUID: "ely-uuid-no-dash", Username: "ElyPlayer", Type: AccountTypeElyBy, OAuth: true})
	if err := svc.saveAccounts(accounts); err != nil {
		t.Fatal(err)
	}

	dataURL, err := svc.GetAccountSkin("ely-uuid-no-dash")
	if err != nil {
		t.Fatalf("GetAccountSkin: %v", err)
	}
	if !strings.HasPrefix(dataURL, "data:image/png;base64,") {
		t.Fatalf("dataURL = %q", dataURL)
	}
	if profileHits != 1 {
		t.Fatalf("profileHits = %d, want 1", profileHits)
	}
	// REQ-022: the session token must never travel with skin requests.
	if authHeaders != 0 {
		t.Fatalf("authorization headers sent = %d, want 0", authHeaders)
	}
}

func TestGetAccountSkinMicrosoftReadsSessionProfile(t *testing.T) {
	allowTestHost(t)
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/session/profile/"):
			if r.Header.Get("Authorization") != "" {
				t.Error("authorization header must not be sent")
			}
			writeProfile(w, t, server.URL+"/skin.png")
		case r.URL.Path == "/skin.png":
			writePNG(w, r)
		default:
			t.Errorf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()
	prevSession := msSessionProfileURL
	msSessionProfileURL = server.URL + "/session/profile/"
	t.Cleanup(func() { msSessionProfileURL = prevSession })

	dir := t.TempDir()
	svc := skinService(dir, msMemoryKeyring{})
	accounts, err := svc.loadAccounts()
	if err != nil {
		t.Fatal(err)
	}
	accounts = append(accounts, Account{UUID: "ms-uuid-1", Username: "SteveMS", Type: AccountTypeMicrosoft})
	if err := svc.saveAccounts(accounts); err != nil {
		t.Fatal(err)
	}

	dataURL, err := svc.GetAccountSkin("ms-uuid-1")
	if err != nil {
		t.Fatalf("GetAccountSkin: %v", err)
	}
	if !strings.HasPrefix(dataURL, "data:image/png;base64,") {
		t.Fatalf("dataURL = %q", dataURL)
	}
}

func TestGetAccountSkinRejectsForeignTextureHost(t *testing.T) {
	profile := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeProfile(w, t, "https://evil.example.com/steal.png")
	}))
	defer profile.Close()
	prevSession := msSessionProfileURL
	msSessionProfileURL = profile.URL + "/session/profile/"
	t.Cleanup(func() { msSessionProfileURL = prevSession })

	dir := t.TempDir()
	svc := skinService(dir, msMemoryKeyring{})
	accounts, err := svc.loadAccounts()
	if err != nil {
		t.Fatal(err)
	}
	accounts = append(accounts, Account{UUID: "ms-foreign-host", Username: "X", Type: AccountTypeMicrosoft})
	if err := svc.saveAccounts(accounts); err != nil {
		t.Fatal(err)
	}

	if _, err := svc.GetAccountSkin("ms-foreign-host"); err == nil {
		t.Fatal("expected foreign texture host to be rejected")
	}
}

func TestGetAccountSkinUnknownAccount(t *testing.T) {
	svc := skinService(t.TempDir(), msMemoryKeyring{})
	if _, err := svc.GetAccountSkin("nope"); err == nil {
		t.Fatal("expected error for unknown account")
	}
}

func TestGetAccountSkinCachesResult(t *testing.T) {
	allowTestHost(t)
	hits := 0
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/session/profile/"):
			hits++
			writeProfile(w, t, server.URL+"/skin.png")
		case r.URL.Path == "/skin.png":
			hits++
			writePNG(w, r)
		default:
			t.Errorf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()
	prevSession := elySessionProfileURL
	elySessionProfileURL = server.URL + "/session/profile/"
	t.Cleanup(func() { elySessionProfileURL = prevSession })

	dir := t.TempDir()
	svc := skinService(dir, msMemoryKeyring{})
	accounts, err := svc.loadAccounts()
	if err != nil {
		t.Fatal(err)
	}
	accounts = append(accounts, Account{UUID: "ely-cache-uuid", Username: "Cached", Type: AccountTypeElyBy})
	if err := svc.saveAccounts(accounts); err != nil {
		t.Fatal(err)
	}
	skinCache.Delete("ely-cache-uuid")
	t.Cleanup(func() { skinCache.Delete("ely-cache-uuid") })

	if _, err := svc.GetAccountSkin("ely-cache-uuid"); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if hits != 2 {
		t.Fatalf("hits after first call = %d, want 2", hits)
	}
	if _, err := svc.GetAccountSkin("ely-cache-uuid"); err != nil {
		t.Fatalf("second call: %v", err)
	}
	if hits != 2 {
		t.Fatalf("hits after cached call = %d, want 2 (no extra requests)", hits)
	}
}
