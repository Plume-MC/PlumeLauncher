package services

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"plumelauncher/internal/auth"
	"plumelauncher/internal/instances"
)

// oauthLaunchServer serves the account-info endpoint only; token polling
// is covered by elyby_oauth_test.go in services_test.
func oauthLaunchServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/account/v1/info" {
			t.Errorf("unexpected path %q", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.Header.Get("Authorization") != "Bearer oauth-launch-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"id":9,"uuid":"ely-launch-uuid","username":"ElyLaunch"}`))
	}))
}

func TestOAuthSessionResolvesForLaunch(t *testing.T) {
	dir := t.TempDir()
	server := oauthLaunchServer(t)
	defer server.Close()
	keys := msMemoryKeyring{}
	svc := &AccountService{
		DataRoot:         dir,
		HTTPClient:       server.Client(),
		Keyring:          keys,
		ElyOAuthBaseURL:  server.URL,
		ElyOAuthClientID: "test-client",
	}
	mgr := instances.NewManager(dir, instances.DefaultLauncherDefaults())
	instSvc := &InstanceService{DataRoot: dir, Manager: mgr}
	created, err := instSvc.CreateInstance("Launch Test", "1.21.4", "vanilla")
	if err != nil {
		t.Fatal(err)
	}
	_ = created

	// Simulate a finished OAuth login: metadata-only account + keyring token.
	accounts, err := svc.loadAccounts()
	if err != nil {
		t.Fatal(err)
	}
	accounts = append(accounts, Account{UUID: "ely-launch-uuid", Username: "ElyLaunch", Type: AccountTypeElyBy, Selected: true, OAuth: true})
	if err := svc.saveAccounts(accounts); err != nil {
		t.Fatal(err)
	}
	if err := keys.Set(auth.SessionKey("ely-launch-uuid"), "oauth-launch-token"); err != nil {
		t.Fatal(err)
	}

	account, token, err := svc.selectedAccountForLaunch()
	if err != nil {
		t.Fatalf("selectedAccountForLaunch: %v", err)
	}
	if account.UUID != "ely-launch-uuid" || account.Type != AccountTypeElyBy || !account.OAuth {
		t.Fatalf("account = %+v", account)
	}
	if token != "oauth-launch-token" {
		t.Fatalf("token = %q", token)
	}
	if !needsAuthlibInjector(account.Type) {
		t.Fatal("oauth ely.by session must use the authlib injector")
	}
}

func TestOAuthSessionExpiredAsksRelogin(t *testing.T) {
	dir := t.TempDir()
	keys := msMemoryKeyring{}
	svc := &AccountService{DataRoot: dir, Keyring: keys}
	accounts, err := svc.loadAccounts()
	if err != nil {
		t.Fatal(err)
	}
	accounts = append(accounts, Account{UUID: "ely-stale-uuid", Username: "ElyStale", Type: AccountTypeElyBy, Selected: true, OAuth: true})
	if err := svc.saveAccounts(accounts); err != nil {
		t.Fatal(err)
	}
	// No token in keyring: launch must ask for re-login, not crash.
	if _, _, err := svc.selectedAccountForLaunch(); err == nil {
		t.Fatal("expected re-login error for missing oauth token")
	}
}
