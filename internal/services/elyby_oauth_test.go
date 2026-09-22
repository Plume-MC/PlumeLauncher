package services_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"plumelauncher/internal/services"
)

// oauthDeviceServer serves the Ely.by device-code endpoints with a scripted
// token-poll behavior driven by pollRespond.
func oauthDeviceServer(t *testing.T, pollRespond func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/oauth2/v1/devicecode":
			_, _ = w.Write([]byte(`{"device_code":"dev-123","user_code":"ABCD-1234","verification_uri":"https://account.ely.by/code","expires_in":600,"interval":5}`))
		case "/api/oauth2/v1/token":
			pollRespond(w, r)
		case "/api/account/v1/info":
			if auth := r.Header.Get("Authorization"); auth != "Bearer oauth-token" && auth != "Bearer oauth-token-2" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			_, _ = w.Write([]byte(`{"id":7,"uuid":"ely-oauth-uuid","username":"ElyOAuthPlayer"}`))
		default:
			t.Errorf("unexpected path %q", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func oauthService(dir string, server *httptest.Server, keyring memoryKeyring) *services.AccountService {
	return &services.AccountService{
		DataRoot:         dir,
		HTTPClient:       server.Client(),
		Keyring:          keyring,
		ElyOAuthBaseURL:  server.URL,
		ElyOAuthClientID: "test-client",
	}
}

func okTokenPoll(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte(`{"access_token":"oauth-token","token_type":"Bearer"}`))
}

func TestStartElyByOAuthDeviceCode(t *testing.T) {
	server := oauthDeviceServer(t, okTokenPoll)
	defer server.Close()
	svc := oauthService(t.TempDir(), server, memoryKeyring{})

	start, err := svc.StartElyByOAuth()
	if err != nil {
		t.Fatalf("StartElyByOAuth: %v", err)
	}
	if start.UserCode != "ABCD-1234" || start.VerificationURI == "" || start.DeviceCode != "dev-123" {
		t.Fatalf("start = %#v", start)
	}
}

func TestFinishElyByOAuthDeviceCode(t *testing.T) {
	server := oauthDeviceServer(t, okTokenPoll)
	defer server.Close()
	keyring := memoryKeyring{}
	svc := oauthService(t.TempDir(), server, keyring)

	account, err := svc.FinishElyByOAuth("dev-123")
	if err != nil {
		t.Fatalf("FinishElyByOAuth: %v", err)
	}
	if account.Type != "ely.by" || account.Username != "ElyOAuthPlayer" || account.UUID != "ely-oauth-uuid" {
		t.Fatalf("account = %#v", account)
	}
	if keyring["ely.by:ely-oauth-uuid"] != "oauth-token" {
		t.Fatalf("access token not in keyring: %#v", keyring)
	}
	// The device-code flow issues no refresh_token (Ely.by DeviceCodeCest:
	// approved responses contain access_token only), so the OAuth access
	// token itself is the keyring session. RefreshElyBy must accept it.
	if _, err := svc.RefreshElyBy(account.UUID); err != nil {
		t.Fatalf("RefreshElyBy (oauth session): %v", err)
	}
}

func TestFinishElyByOAuthRejectsPending(t *testing.T) {
	server := oauthDeviceServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"authorization_pending"}`))
	})
	defer server.Close()
	svc := oauthService(t.TempDir(), server, memoryKeyring{})

	if _, err := svc.FinishElyByOAuth("dev-123"); err == nil {
		t.Fatal("expected pending error")
	}
}

func TestFinishElyByOAuthRejectsDenied(t *testing.T) {
	server := oauthDeviceServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"access_denied"}`))
	})
	defer server.Close()
	svc := oauthService(t.TempDir(), server, memoryKeyring{})

	if _, err := svc.FinishElyByOAuth("dev-123"); err == nil || !strings.Contains(strings.ToLower(err.Error()), "denied") {
		t.Fatalf("error = %v, want denied", err)
	}
}

func TestCancelElyByOAuthIsNoop(t *testing.T) {
	server := oauthDeviceServer(t, okTokenPoll)
	defer server.Close()
	svc := oauthService(t.TempDir(), server, memoryKeyring{})

	// Device-code sessions live on the Ely.by website; cancelling locally
	// must not touch the network or stored accounts.
	if err := svc.CancelElyByOAuth("dev-123"); err != nil {
		t.Fatalf("CancelElyByOAuth: %v", err)
	}
}

func TestElyByOAuthFailsClosedWhenKeyringUnavailable(t *testing.T) {
	server := oauthDeviceServer(t, okTokenPoll)
	defer server.Close()
	svc := oauthService(t.TempDir(), server, nil)
	svc.Keyring = unavailableKeyring{}

	if _, err := svc.FinishElyByOAuth("dev-123"); err == nil || !strings.Contains(err.Error(), "secure storage") {
		t.Fatalf("error = %v, want secure storage failure", err)
	}
}

func TestFinishElyByOAuthLeavesNoTokenOnProfileFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/oauth2/v1/devicecode":
			_, _ = w.Write([]byte(`{"device_code":"dev-123","user_code":"ABCD-1234","verification_uri":"https://account.ely.by/code","expires_in":600,"interval":5}`))
		case "/api/oauth2/v1/token":
			okTokenPoll(w, r)
		case "/api/account/v1/info":
			w.WriteHeader(http.StatusUnauthorized)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	keyring := memoryKeyring{}
	svc := oauthService(t.TempDir(), server, keyring)

	if _, err := svc.FinishElyByOAuth("dev-123"); err == nil {
		t.Fatal("expected profile failure")
	}
	if len(keyring) != 0 {
		t.Fatalf("orphaned token remains: %#v", keyring)
	}
}
