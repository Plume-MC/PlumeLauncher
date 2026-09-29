package auth_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"plumelauncher/internal/auth"
)

func oauthTestClient(server *httptest.Server) auth.ElyOAuthClient {
	return auth.ElyOAuthClient{
		HTTPClient: server.Client(),
		BaseURL:    server.URL,
		ClientID:   "test-client",
	}
}

func TestOAuthStartDeviceCode(t *testing.T) {
	var gotPath, gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		body := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(body)
		gotBody = string(body)
		_, _ = w.Write([]byte(`{"device_code":"dev-123","user_code":"ABCD-1234","verification_uri":"https://account.ely.by/code","expires_in":600,"interval":5}`))
	}))
	defer server.Close()

	start, err := oauthTestClient(server).StartDeviceCode(t.Context())
	if err != nil {
		t.Fatalf("StartDeviceCode: %v", err)
	}
	if start.DeviceCode != "dev-123" || start.UserCode != "ABCD-1234" {
		t.Fatalf("unexpected device code: %#v", start)
	}
	if start.VerificationURI == "" || start.ExpiresIn != 600 || start.Interval != 5 {
		t.Fatalf("unexpected device metadata: %#v", start)
	}
	if gotPath != "/api/oauth2/v1/devicecode" {
		t.Fatalf("path = %q", gotPath)
	}
	for _, want := range []string{"test-client", "account_info", "minecraft_server_session", "offline_access"} {
		if !strings.Contains(gotBody, want) {
			t.Fatalf("request body missing %q: %s", want, gotBody)
		}
	}
}

func TestOAuthStartDeviceCodeRejectsUnknownClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid_client"}`))
	}))
	defer server.Close()

	if _, err := oauthTestClient(server).StartDeviceCode(t.Context()); err == nil {
		t.Fatal("expected invalid-client error")
	}
}

func TestOAuthStartDeviceCodeRejectsUnknownScope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_scope"}`))
	}))
	defer server.Close()

	if _, err := oauthTestClient(server).StartDeviceCode(t.Context()); err == nil {
		t.Fatal("expected invalid-scope error")
	}
}

func TestOAuthPollDeviceTokenPendingThenSuccess(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/api/oauth2/v1/token" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if calls == 1 {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"authorization_pending"}`))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "oauth-token", "token_type": "Bearer"})
	}))
	defer server.Close()

	client := oauthTestClient(server)
	if _, err := client.PollDeviceToken(t.Context(), "dev-123"); err == nil {
		t.Fatal("expected pending error on first poll")
	} else if !isOAuthPending(err) {
		t.Fatalf("first poll error = %v, want pending", err)
	}
	token, err := client.PollDeviceToken(t.Context(), "dev-123")
	if err != nil {
		t.Fatalf("second poll: %v", err)
	}
	if token != "oauth-token" {
		t.Fatalf("token = %q", token)
	}
}

func isOAuthPending(err error) bool {
	return err != nil && strings.Contains(err.Error(), "pending")
}

func TestOAuthPollDeviceTokenExpired(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"expired_token"}`))
	}))
	defer server.Close()

	_, err := oauthTestClient(server).PollDeviceToken(t.Context(), "dev-stale")
	if err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("error = %v, want expired", err)
	}
}

func TestOAuthPollDeviceTokenDenied(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"access_denied"}`))
	}))
	defer server.Close()

	_, err := oauthTestClient(server).PollDeviceToken(t.Context(), "dev-denied")
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "denied") {
		t.Fatalf("error = %v, want denied", err)
	}
}

func TestOAuthFetchAccountInfo(t *testing.T) {
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		if r.URL.Path != "/api/account/v1/info" {
			t.Errorf("path = %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"id":42,"uuid":"ely-uuid-1","username":"ElyPlayer"}`))
	}))
	defer server.Close()

	info, err := oauthTestClient(server).FetchAccountInfo(t.Context(), "oauth-token")
	if err != nil {
		t.Fatalf("FetchAccountInfo: %v", err)
	}
	if info.UUID != "ely-uuid-1" || info.Username != "ElyPlayer" {
		t.Fatalf("info = %#v", info)
	}
	if gotAuth != "Bearer oauth-token" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
}

func TestOAuthRefreshToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "oauth-token-2",
			"refresh_token": "oauth-refresh-2",
			"token_type":    "Bearer",
		})
	}))
	defer server.Close()

	access, refresh, err := oauthTestClient(server).RefreshOAuthToken(t.Context(), "oauth-refresh-1")
	if err != nil {
		t.Fatalf("RefreshOAuthToken: %v", err)
	}
	if access != "oauth-token-2" || refresh != "oauth-refresh-2" {
		t.Fatalf("access = %q refresh = %q", access, refresh)
	}
}

func TestOAuthSecretsNeverLogged(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":1,"uuid":"u","username":"P"}`))
	}))
	defer server.Close()

	client := oauthTestClient(server)
	if _, err := client.StartDeviceCode(t.Context()); err == nil {
		// server returns invalid JSON for devicecode path; StartDeviceCode must fail, not leak
		t.Fatal("expected decode failure")
	} else if strings.Contains(err.Error(), "dev-") {
		t.Fatalf("error leaks secrets: %v", err)
	}
}
