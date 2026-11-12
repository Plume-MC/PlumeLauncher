package auth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func testContext() context.Context { return context.Background() }

type memoryKeyring struct{ data map[string]string }

func newMemoryKeyring() *memoryKeyring { return &memoryKeyring{data: map[string]string{}} }

func (m *memoryKeyring) Set(key, value string) error { m.data[key] = value; return nil }
func (m *memoryKeyring) Get(key string) (string, error) {
	v, ok := m.data[key]
	if !ok {
		return "", ErrTokenNotFound
	}
	return v, nil
}
func (m *memoryKeyring) Delete(key string) error { delete(m.data, key); return nil }

func overrideEntraOAuthURL(t *testing.T, url string) {
	t.Helper()
	prev := entraTokenURL
	entraTokenURL = url
	t.Cleanup(func() { entraTokenURL = prev })
}

func TestMakePKCEChallengeMatchesVerifier(t *testing.T) {
	verifier := generateRandomHex(64)
	if len(verifier) != 64 {
		t.Fatalf("verifier must be 64 hex chars, got %q", verifier)
	}
	h := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(h[:])
	if len(challenge) != 43 {
		t.Fatalf("S256 challenge must be 43 chars, got %d", len(challenge))
	}
}

func mockEntraServer(t *testing.T, check func(r *http.Request, form string)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if check != nil {
			check(r, string(body))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token":  "ms-access",
			"refresh_token": "ms-refresh",
			"expires_in":    3600,
		})
	}))
}

func TestEntraClientIDIsSet(t *testing.T) {
	if EntraClientID == "" || len(EntraClientID) < 8 {
		t.Fatal("EntraClientID must be the registered app id")
	}
}

func TestOAuthRefreshRejectsUpstreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid_grant"}`))
	}))
	defer srv.Close()
	overrideEntraOAuthURL(t, srv.URL)

	if _, err := EntraRefresh(testContext(), "bad-refresh"); err == nil {
		t.Fatal("expected error for 400 response")
	}
}

func TestEntraXboxChainRoundTrip(t *testing.T) {
	tokenJSON := func(token string) string {
		xui, _ := json.Marshal([]map[string]string{{"uhs": "userhash123"}})
		return `{"Token":` + strconv.Quote(token) + `,"DisplayClaims":{"xui":` + string(xui) + `}}`
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/user/authenticate", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "d=ms-access") {
			t.Errorf("RPS ticket malformed: %s", body)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(tokenJSON("xbox-tok")))
	})
	mux.HandleFunc("/xsts/authorize", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(tokenJSON("xsts-tok")))
	})
	mux.HandleFunc("/authentication/login_with_xbox", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"access_token":"mc-access","expires_in":3600}`))
	})
	mux.HandleFunc("/entitlements/mcstore", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"items":[{"name":"game_minecraft"}]}`))
	})
	mux.HandleFunc("/minecraft/profile", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer mc-access" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"player-uuid-1","name":"Steve","skins":[],"capes":[]}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	overrideEntraChainURLs(t, srv.URL+"/user/authenticate", srv.URL+"/xsts/authorize",
		srv.URL+"/authentication/login_with_xbox", srv.URL+"/entitlements/mcstore")
	prevProf := mcProfileURL
	mcProfileURL = srv.URL + "/minecraft/profile"
	t.Cleanup(func() { mcProfileURL = prevProf })

	creds, err := EntraCompleteLogin(testContext(), EntraToken{
		AccessToken:  "ms-access",
		RefreshToken: "ms-refresh",
		ExpiresAt:    time.Now().Add(time.Hour),
	}, newMemoryKeyring())
	if err != nil {
		t.Fatalf("EntraCompleteLogin: %v", err)
	}
	if creds.UUID != "player-uuid-1" || creds.Username != "Steve" || creds.AccessToken != "mc-access" {
		t.Fatalf("unexpected creds: %+v", creds)
	}
}

func TestRefreshMicrosoftRotatesSession(t *testing.T) {
	entraMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"access_token":"ms-access","refresh_token":"ms-refresh-new","expires_in":3600}`))
	})
	chainRoundTrip(t)

	store := newMemoryKeyring()
	creds, err := RefreshMicrosoft(testContext(), "old-refresh", store)
	if err != nil {
		t.Fatalf("RefreshMicrosoft: %v", err)
	}
	if creds.AccessToken != "mc-access" || creds.UUID != "player-uuid-1" {
		t.Fatalf("unexpected creds: %+v", creds)
	}
	if got, _ := store.Get(MicrosoftRefreshKey("player-uuid-1")); got != "ms-refresh-new" {
		t.Fatalf("rotated refresh token must be in keyring, got %q", got)
	}
}

// chainRoundTrip points the Xbox/Minecraft chain at a minimal fake.
func chainRoundTrip(t *testing.T) {
	t.Helper()
	tokenJSON := `{"Token":"xsts-tok","DisplayClaims":{"xui":[{"uhs":"userhash123"}]}}`
	mux := http.NewServeMux()
	mux.HandleFunc("/user/authenticate", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(tokenJSON))
	})
	mux.HandleFunc("/xsts/authorize", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(tokenJSON))
	})
	mux.HandleFunc("/authentication/login_with_xbox", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"access_token":"mc-access","expires_in":3600}`))
	})
	mux.HandleFunc("/entitlements/mcstore", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"items":[{"name":"game_minecraft"}]}`))
	})
	mux.HandleFunc("/minecraft/profile", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"player-uuid-1","name":"Steve","skins":[],"capes":[]}`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	overrideEntraChainURLs(t, srv.URL+"/user/authenticate", srv.URL+"/xsts/authorize",
		srv.URL+"/authentication/login_with_xbox", srv.URL+"/entitlements/mcstore")
	prevProf := mcProfileURL
	mcProfileURL = srv.URL + "/minecraft/profile"
	t.Cleanup(func() { mcProfileURL = prevProf })
}

func TestEnsureValidMicrosoftTokenBuffer(t *testing.T) {
	entraMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"access_token":"ms-access","refresh_token":"ms-refresh-new","expires_in":3600}`))
	})
	chainRoundTrip(t)
	store := newMemoryKeyring()

	// Far-future expiry: no refresh, empty tokens back.
	access, refresh, _, err := EnsureValidMicrosoftToken(testContext(), "old", time.Now().Add(time.Hour), store)
	if err != nil || access != "" || refresh != "" {
		t.Fatalf("valid token must not refresh: %q %q %v", access, refresh, err)
	}
	// Near expiry: refresh happens.
	access, refresh, exp, err := EnsureValidMicrosoftToken(testContext(), "old", time.Now().Add(time.Minute), store)
	if err != nil || access != "mc-access" || refresh != "ms-refresh-new" {
		t.Fatalf("near-expiry token must refresh: %q %q %v", access, refresh, err)
	}
	if time.Until(exp) < time.Minute {
		t.Fatal("refreshed expiry must be in the future")
	}
}

func TestRefreshMicrosoftInvalidGrantStaysTyped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid_grant","error_description":"token expired","access_token":"must-never-leak"}`))
	}))
	defer srv.Close()
	overrideEntraOAuthURL(t, srv.URL)

	_, err := RefreshMicrosoft(testContext(), "dead-refresh", newMemoryKeyring())
	if err == nil || !errors.Is(err, ErrInvalidGrant) {
		t.Fatalf("invalid_grant must match ErrInvalidGrant, got %v", err)
	}
	if strings.Contains(err.Error(), "must-never-leak") {
		t.Fatalf("refresh error must not echo response body: %v", err)
	}
}

func TestRefreshMicrosoftKeepsTokenWhenRotationEmpty(t *testing.T) {
	entraMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"access_token":"ms-access","expires_in":3600}`))
	})
	chainRoundTrip(t)

	store := newMemoryKeyring()
	creds, err := RefreshMicrosoft(testContext(), "stable-refresh", store)
	if err != nil {
		t.Fatalf("RefreshMicrosoft: %v", err)
	}
	if creds.RefreshToken != "stable-refresh" {
		t.Fatalf("empty rotation must keep old token, got %q", creds.RefreshToken)
	}
	if got, _ := store.Get(MicrosoftRefreshKey(creds.UUID)); got != "stable-refresh" {
		t.Fatalf("keyring must keep old token, got %q", got)
	}
}

func TestRefreshMicrosoftDedupesConcurrentCalls(t *testing.T) {
	entraMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"access_token":"ms-access","refresh_token":"ms-refresh-new","expires_in":3600}`))
	})
	chainRoundTrip(t)
	var hits atomic.Int32

	const callers = 5
	results := make([]*MicrosoftCredentials, callers)
	errs := make([]error, callers)
	var wg sync.WaitGroup
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i], errs[i] = RefreshMicrosoft(testContext(), "shared-refresh", newMemoryKeyring())
		}(i)
	}
	wg.Wait()
	for i := 0; i < callers; i++ {
		if errs[i] != nil || results[i].AccessToken != "mc-access" {
			t.Fatalf("caller %d: %+v %v", i, results[i], errs[i])
		}
	}
	if hits.Load() != 0 {
		t.Fatal("no extra oauth hits expected outside the mock")
	}
}
