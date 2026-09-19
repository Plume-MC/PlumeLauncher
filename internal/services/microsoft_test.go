package services

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"plumelauncher/internal/auth"
)

type msMemoryKeyring map[string]string

func (k msMemoryKeyring) Set(key, value string) error { k[key] = value; return nil }
func (k msMemoryKeyring) Get(key string) (string, error) {
	v, ok := k[key]
	if !ok {
		return "", auth.ErrTokenNotFound
	}
	return v, nil
}
func (k msMemoryKeyring) Delete(key string) error { delete(k, key); return nil }

func testMicrosoftCreds() *auth.MicrosoftCredentials {
	return &auth.MicrosoftCredentials{
		AccessToken:  "mc-access-token",
		RefreshToken: "ms-refresh-token",
		ExpiresAt:    time.Now().Add(time.Hour),
		UUID:         "player-uuid-1",
		Username:     "Steve",
	}
}

func TestUpsertMicrosoftAccountIsMetadataOnly(t *testing.T) {
	dir := t.TempDir()
	keys := msMemoryKeyring{}
	svc := &AccountService{DataRoot: dir, Keyring: keys}

	acc, err := svc.upsertMicrosoftAccount(testMicrosoftCreds())
	if err != nil {
		t.Fatalf("upsertMicrosoftAccount: %v", err)
	}
	if acc.Type != AccountTypeMicrosoft || !acc.Selected {
		t.Fatalf("account = %+v", acc)
	}
	if keys[auth.MicrosoftRefreshKey("player-uuid-1")] != "ms-refresh-token" {
		t.Fatal("refresh token must be in keyring")
	}
	if keys[auth.MicrosoftAccessKey("player-uuid-1")] != "mc-access-token" {
		t.Fatal("access token must be in keyring")
	}
	if _, ok := keys[auth.MicrosoftExpiryKey("player-uuid-1")]; !ok {
		t.Fatal("expiry must be in keyring")
	}
	raw, _ := os.ReadFile(filepath.Join(dir, "accounts.json"))
	for _, secret := range []string{"mc-access-token", "ms-refresh-token", "accessToken", "refreshToken"} {
		if strings.Contains(string(raw), secret) {
			t.Fatalf("accounts.json contains %q", secret)
		}
	}
}

func TestUpsertMicrosoftAccountDeselectsOthers(t *testing.T) {
	dir := t.TempDir()
	svc := &AccountService{DataRoot: dir, Keyring: msMemoryKeyring{}}
	if _, err := svc.CreateOffline("Old"); err != nil {
		t.Fatal(err)
	}
	acc, err := svc.upsertMicrosoftAccount(testMicrosoftCreds())
	if err != nil {
		t.Fatal(err)
	}
	if !acc.Selected {
		t.Fatal("microsoft account must be selected")
	}
	accounts, _ := svc.ListAccounts()
	for _, a := range accounts {
		if a.Type == AccountTypeOffline && a.Selected {
			t.Fatal("offline account must be deselected")
		}
	}
}

func TestRefreshMicrosoftTokenFailsClosedWithoutKeyring(t *testing.T) {
	dir := t.TempDir()
	keys := msMemoryKeyring{}
	svc := &AccountService{DataRoot: dir, Keyring: keys}
	if _, err := svc.upsertMicrosoftAccount(testMicrosoftCreds()); err != nil {
		t.Fatal(err)
	}
	delete(keys, auth.MicrosoftRefreshKey("player-uuid-1"))
	if _, err := svc.RefreshMicrosoftToken("player-uuid-1"); err == nil {
		t.Fatal("expected session-expired error without keyring entry")
	} else if !strings.Contains(err.Error(), "sign in again") {
		t.Fatalf("error must request re-auth, got %v", err)
	}
}

func TestRefreshMicrosoftTokenRejectsUnknownAccount(t *testing.T) {
	svc := &AccountService{DataRoot: t.TempDir(), Keyring: msMemoryKeyring{}}
	if _, err := svc.RefreshMicrosoftToken("nope"); err == nil {
		t.Fatal("expected not-found error")
	}
}

func TestDeleteAccountDispatchesMicrosoft(t *testing.T) {
	dir := t.TempDir()
	keys := msMemoryKeyring{}
	svc := &AccountService{DataRoot: dir, Keyring: keys}
	if _, err := svc.upsertMicrosoftAccount(testMicrosoftCreds()); err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteAccount("player-uuid-1"); err != nil {
		t.Fatalf("DeleteAccount: %v", err)
	}
	accounts, _ := svc.ListAccounts()
	if len(accounts) != 0 {
		t.Fatalf("accounts = %+v", accounts)
	}
	for _, k := range []string{
		auth.MicrosoftRefreshKey("player-uuid-1"),
		auth.MicrosoftAccessKey("player-uuid-1"),
		auth.MicrosoftExpiryKey("player-uuid-1"),
	} {
		if _, ok := keys[k]; ok {
			t.Fatalf("keyring key %q must be cleaned", k)
		}
	}
}

func TestSelectedMicrosoftAccountReturnsKeyringToken(t *testing.T) {
	dir := t.TempDir()
	svc := &AccountService{DataRoot: dir, Keyring: msMemoryKeyring{}}
	if _, err := svc.upsertMicrosoftAccount(testMicrosoftCreds()); err != nil {
		t.Fatal(err)
	}
	account, token, err := svc.selectedAccount()
	if err != nil {
		t.Fatalf("selectedAccount: %v", err)
	}
	if account.Type != AccountTypeMicrosoft || token != "mc-access-token" {
		t.Fatalf("account = %+v token = %q", account, token)
	}
}

func TestSelectedMicrosoftAccountFailsClosedWithoutSession(t *testing.T) {
	dir := t.TempDir()
	keys := msMemoryKeyring{}
	svc := &AccountService{DataRoot: dir, Keyring: keys}
	if _, err := svc.upsertMicrosoftAccount(testMicrosoftCreds()); err != nil {
		t.Fatal(err)
	}
	delete(keys, auth.MicrosoftRefreshKey("player-uuid-1"))
	if _, _, err := svc.selectedAccount(); err == nil {
		t.Fatal("expected session-expired error")
	}
}

func TestLoginMicrosoftRequiresApp(t *testing.T) {
	svc := &AccountService{DataRoot: t.TempDir(), Keyring: msMemoryKeyring{}}
	if _, err := svc.LoginMicrosoft(); err == nil {
		t.Fatal("expected error when App is not wired")
	}
}

func TestNeedsAuthlibInjectorOnlyElyBy(t *testing.T) {
	if !needsAuthlibInjector(AccountTypeElyBy) {
		t.Fatal("ely.by must use authlib-injector")
	}
	for _, typ := range []string{AccountTypeOffline, AccountTypeMicrosoft, "", "mojang"} {
		if needsAuthlibInjector(typ) {
			t.Fatalf("account type %q must not use authlib-injector", typ)
		}
	}
}

func TestRefreshMicrosoftErrorClassification(t *testing.T) {
	expired := refreshMicrosoftError(auth.ErrInvalidGrant)
	if !strings.Contains(expired.Error(), "sign in again") {
		t.Fatalf("invalid grant must request re-login, got %v", expired)
	}
	transient := refreshMicrosoftError(auth.ErrTokenNotFound)
	if strings.Contains(transient.Error(), "sign in again") || !strings.Contains(transient.Error(), "retry") {
		t.Fatalf("transient failure must be retryable, got %v", transient)
	}
}

func TestParseMicrosoftCallback(t *testing.T) {
	req := httptest.NewRequest("GET", "/callback?code=auth-code-123", nil)
	code, err := parseMicrosoftCallback(req)
	if err != nil || code != "auth-code-123" {
		t.Fatalf("code = %q err = %v", code, err)
	}

	req = httptest.NewRequest("GET", "/callback?error=access_denied&error_description=denied+by+user", nil)
	if _, err := parseMicrosoftCallback(req); err == nil || !strings.Contains(err.Error(), "denied by user") {
		t.Fatalf("must prefer error_description, got %v", err)
	}

	req = httptest.NewRequest("GET", "/callback", nil)
	if _, err := parseMicrosoftCallback(req); err == nil || !strings.Contains(err.Error(), "no code in callback") {
		t.Fatalf("empty callback must fail fast, got %v", err)
	}
}
