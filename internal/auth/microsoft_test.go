package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func overrideOAuthURL(t *testing.T, url string) {
	t.Helper()
	prev := msOAuthTokenURL
	msOAuthTokenURL = url
	t.Cleanup(func() { msOAuthTokenURL = prev })
}

func TestGenerateKeyProducesP256Coordinates(t *testing.T) {
	key, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	if key.Key == nil || key.X == "" || key.Y == "" {
		t.Fatal("key, X and Y must be populated")
	}
	if len(key.X) != 43 || len(key.Y) != 43 {
		t.Fatalf("X/Y must be 32-byte base64url (43 chars), got %d/%d", len(key.X), len(key.Y))
	}
}

func TestMakePKCEChallengeMatchesVerifier(t *testing.T) {
	verifier, challenge := makePKCE()
	if len(verifier) != 64 {
		t.Fatalf("verifier must be 64 hex chars, got %q", verifier)
	}
	h := sha256.Sum256([]byte(verifier))
	if challenge != base64.RawURLEncoding.EncodeToString(h[:]) {
		t.Fatal("challenge must be base64url(sha256(verifier))")
	}
}

func mockOAuthServer(t *testing.T, check func(r *http.Request, form string)) *httptest.Server {
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

func TestOAuthTokenExchangePostsAuthorizationCodeGrant(t *testing.T) {
	var sawForm, sawCT string
	srv := mockOAuthServer(t, func(r *http.Request, form string) {
		sawForm, sawCT = form, r.Header.Get("Content-Type")
	})
	defer srv.Close()
	overrideOAuthURL(t, srv.URL)

	resp, err := OAuthTokenExchange("auth-code", "verifier-abc")
	if err != nil {
		t.Fatalf("OAuthTokenExchange: %v", err)
	}
	if resp.AccessToken != "ms-access" || resp.RefreshToken != "ms-refresh" || resp.ExpiresIn != 3600 {
		t.Fatalf("unexpected token response: %+v", resp)
	}
	for _, want := range []string{"grant_type=authorization_code", "code=auth-code", "code_verifier=verifier-abc"} {
		if !strings.Contains(sawForm, want) {
			t.Fatalf("form missing %q: %s", want, sawForm)
		}
	}
	if sawCT != "application/x-www-form-urlencoded" {
		t.Fatalf("content type = %q", sawCT)
	}
}

func TestOAuthRefreshPostsRefreshGrant(t *testing.T) {
	var sawForm string
	srv := mockOAuthServer(t, func(r *http.Request, form string) { sawForm = form })
	defer srv.Close()
	overrideOAuthURL(t, srv.URL)

	resp, err := OAuthRefresh("old-refresh")
	if err != nil {
		t.Fatalf("OAuthRefresh: %v", err)
	}
	if resp.RefreshToken == "" {
		t.Fatal("expected refreshed token")
	}
	for _, want := range []string{"grant_type=refresh_token", "refresh_token=old-refresh"} {
		if !strings.Contains(sawForm, want) {
			t.Fatalf("form missing %q: %s", want, sawForm)
		}
	}
}

func TestOAuthTokenExchangeRejectsUpstreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid_grant"}`))
	}))
	defer srv.Close()
	overrideOAuthURL(t, srv.URL)

	if _, err := OAuthTokenExchange("bad-code", "v"); err == nil {
		t.Fatal("expected error for 400 response")
	}
	if _, err := OAuthRefresh("bad-refresh"); err == nil {
		t.Fatal("expected error for 400 response")
	}
}
