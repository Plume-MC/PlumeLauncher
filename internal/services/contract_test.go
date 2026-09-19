package services_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"plumelauncher/internal/instances"
	"plumelauncher/internal/services"
)

type memoryKeyring map[string]string

type unavailableKeyring struct{}

func (unavailableKeyring) Set(string, string) error { return fmt.Errorf("keyring unavailable") }
func (unavailableKeyring) Get(string) (string, error) {
	return "", fmt.Errorf("keyring unavailable")
}
func (unavailableKeyring) Delete(string) error { return fmt.Errorf("keyring unavailable") }

func (k memoryKeyring) Set(key, value string) error { k[key] = value; return nil }
func (k memoryKeyring) Get(key string) (string, error) {
	value, ok := k[key]
	if !ok {
		return "", fmt.Errorf("not found")
	}
	return value, nil
}
func (k memoryKeyring) Delete(key string) error { delete(k, key); return nil }

func TestAccountServiceCreateOffline(t *testing.T) {
	dir := t.TempDir()
	svc := &services.AccountService{DataRoot: dir}

	acc, err := svc.CreateOffline("TestPlayer")
	if err != nil {
		t.Fatalf("CreateOffline: %v", err)
	}
	if acc.Username != "TestPlayer" {
		t.Errorf("Username = %q", acc.Username)
	}
	if acc.Type != "offline" {
		t.Errorf("Type = %q", acc.Type)
	}
	if acc.UUID == "" {
		t.Error("UUID is empty")
	}
}

func TestAccountServiceCreateDuplicate(t *testing.T) {
	dir := t.TempDir()
	svc := &services.AccountService{DataRoot: dir}

	svc.CreateOffline("TestPlayer")
	_, err := svc.CreateOffline("TestPlayer")
	if err == nil {
		t.Fatal("expected error for duplicate")
	}
}

func TestAccountServiceListAccounts(t *testing.T) {
	dir := t.TempDir()
	svc := &services.AccountService{DataRoot: dir}

	svc.CreateOffline("Player1")
	svc.CreateOffline("Player2")

	accounts, err := svc.ListAccounts()
	if err != nil {
		t.Fatalf("ListAccounts: %v", err)
	}
	if len(accounts) != 2 {
		t.Errorf("ListAccounts len = %d, want 2", len(accounts))
	}
}

func TestAccountServiceDeleteOffline(t *testing.T) {
	dir := t.TempDir()
	svc := &services.AccountService{DataRoot: dir}
	account, err := svc.CreateOffline("Player1")
	if err != nil {
		t.Fatalf("CreateOffline: %v", err)
	}
	if _, err := svc.CreateOffline("Player2"); err != nil {
		t.Fatalf("CreateOffline: %v", err)
	}
	if err := svc.DeleteAccount(account.UUID); err != nil {
		t.Fatalf("DeleteAccount: %v", err)
	}
	accounts, err := svc.ListAccounts()
	if err != nil {
		t.Fatalf("ListAccounts: %v", err)
	}
	if len(accounts) != 1 || accounts[0].Username != "Player2" {
		t.Fatalf("ListAccounts = %#v, want only Player2", accounts)
	}
}

func TestAccountServiceDeleteLastAccountFails(t *testing.T) {
	dir := t.TempDir()
	svc := &services.AccountService{DataRoot: dir}
	account, err := svc.CreateOffline("Player1")
	if err != nil {
		t.Fatalf("CreateOffline: %v", err)
	}
	if err := svc.DeleteAccount(account.UUID); err == nil || !strings.Contains(err.Error(), "last account") {
		t.Fatalf("DeleteAccount = %v, want last-account conflict", err)
	}
}

func TestAccountServiceElyByLifecycle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/invalidate" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		_, _ = w.Write([]byte(`{"accessToken":"session-token","selectedProfile":{"id":"ely-uuid","name":"ElyPlayer"}}`))
	}))
	defer server.Close()
	keyring := memoryKeyring{}
	svc := &services.AccountService{DataRoot: t.TempDir(), HTTPClient: server.Client(), ElyByURL: server.URL + "/auth/", Keyring: keyring}
	account, err := svc.LoginElyBy("player", "password")
	if err != nil {
		t.Fatalf("LoginElyBy: %v", err)
	}
	if account.Type != "ely.by" || keyring["ely.by:ely-uuid"] != "session-token" {
		t.Fatalf("unexpected account or token: %#v, %#v", account, keyring)
	}
	if _, err := svc.RefreshElyBy(account.UUID); err != nil {
		t.Fatalf("RefreshElyBy: %v", err)
	}
	if _, err := svc.CreateOffline("Fallback"); err != nil {
		t.Fatalf("CreateOffline: %v", err)
	}
	if err := svc.DeleteAccount(account.UUID); err != nil {
		t.Fatalf("DeleteAccount: %v", err)
	}
	if len(keyring) != 0 {
		t.Fatal("logout left token in keyring")
	}
}

func TestAccountServiceLoginRemovesTokenWhenPersistenceFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"accessToken":"session-token","selectedProfile":{"id":"ely-uuid","name":"ElyPlayer"}}`))
	}))
	defer server.Close()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "accounts.json"), []byte("not-json"), 0o600); err != nil {
		t.Fatal(err)
	}
	keyring := memoryKeyring{}
	svc := &services.AccountService{DataRoot: dir, HTTPClient: server.Client(), ElyByURL: server.URL + "/auth/", Keyring: keyring}
	if _, err := svc.LoginElyBy("player", "password"); err == nil {
		t.Fatal("expected account persistence failure")
	}
	if len(keyring) != 0 {
		t.Fatalf("orphaned token remains: %#v", keyring)
	}
}

func TestAccountServiceFailsClosedWhenKeyringUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"accessToken":"session-token","selectedProfile":{"id":"ely-uuid","name":"ElyPlayer"}}`))
	}))
	defer server.Close()
	dir := t.TempDir()
	svc := &services.AccountService{DataRoot: dir, HTTPClient: server.Client(), ElyByURL: server.URL + "/auth/", Keyring: unavailableKeyring{}}
	if _, err := svc.LoginElyBy("player", "password"); err == nil || !strings.Contains(err.Error(), "secure session storage") {
		t.Fatalf("error = %v, want secure storage failure", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "accounts.json")); !os.IsNotExist(err) {
		t.Fatalf("accounts file exists after keyring failure: %v", err)
	}
}

func TestInstanceServiceCreate(t *testing.T) {
	dir := t.TempDir()
	defaults := instances.DefaultLauncherDefaults()
	mgr := instances.NewManager(dir, defaults)
	svc := &services.InstanceService{DataRoot: dir, Manager: mgr}

	inst, err := svc.CreateInstance("Test Instance", "1.21.4", "vanilla")
	if err != nil {
		t.Fatalf("CreateInstance: %v", err)
	}
	if inst.Name != "Test Instance" {
		t.Errorf("Name = %q", inst.Name)
	}
}

func TestInstanceServiceCreateValidation(t *testing.T) {
	dir := t.TempDir()
	mgr := instances.NewManager(dir, instances.DefaultLauncherDefaults())
	svc := &services.InstanceService{DataRoot: dir, Manager: mgr}

	_, err := svc.CreateInstance("", "1.21.4", "vanilla")
	if err == nil {
		t.Fatal("expected error for empty name")
	}

	_, err = svc.CreateInstance("Test", "", "vanilla")
	if err == nil {
		t.Fatal("expected error for empty version")
	}

	_, err = svc.CreateInstance("Test", "1.21.4", "invalid")
	if err == nil {
		t.Fatal("expected error for invalid loader")
	}
}

func TestInstanceServiceSettingsValidation(t *testing.T) {
	dir := t.TempDir()
	mgr := instances.NewManager(dir, instances.DefaultLauncherDefaults())
	svc := &services.InstanceService{DataRoot: dir, Manager: mgr}

	inst, _ := svc.CreateInstance("Test", "1.21.4", "vanilla")

	// Too low RAM
	err := svc.UpdateInstanceSettings(inst.ID, instances.Settings{
		MinRamMB: instances.IntPtr(100),
	})
	if err == nil {
		t.Fatal("expected error for low RAM")
	}

	// Too high RAM
	err = svc.UpdateInstanceSettings(inst.ID, instances.Settings{
		MaxRamMB: instances.IntPtr(100000),
	})
	if err == nil {
		t.Fatal("expected error for high RAM")
	}
}

func TestSystemServiceGetSettings(t *testing.T) {
	dir := t.TempDir()
	svc := &services.SystemService{DataRoot: dir, Defaults: instances.DefaultLauncherDefaults()}

	settings := svc.GetSettings()
	if settings.Theme != "dark" {
		t.Errorf("Theme = %q", settings.Theme)
	}
}

func TestSystemServiceUpdateSettings(t *testing.T) {
	dir := t.TempDir()
	svc := &services.SystemService{DataRoot: dir, Defaults: instances.DefaultLauncherDefaults()}

	newSettings := instances.DefaultLauncherDefaults()
	newSettings.DefaultMinRamMB = 2048

	err := svc.UpdateSettings(newSettings)
	if err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}

	if svc.GetSettings().DefaultMinRamMB != 2048 {
		t.Errorf("DefaultMinRamMB = %d, want 2048", svc.GetSettings().DefaultMinRamMB)
	}
	loaded, err := instances.LoadConfig(dir)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if loaded.DefaultMinRamMB != 2048 {
		t.Errorf("persisted DefaultMinRamMB = %d, want 2048", loaded.DefaultMinRamMB)
	}
}

func TestSystemServiceUpdateDataRootRejectsEmptyPath(t *testing.T) {
	svc := &services.SystemService{DataRoot: t.TempDir()}
	if err := svc.UpdateDataRoot(""); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestSystemServiceUsesSingleDataRoot(t *testing.T) {
	dataRoot := t.TempDir()
	svc := &services.SystemService{DataRoot: dataRoot}
	logDir := filepath.Join(dataRoot, "logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := svc.GetAppRoot(); got != dataRoot {
		t.Fatalf("GetAppRoot = %q, want %q", got, dataRoot)
	}
	if svc.GetDataRoot() != dataRoot {
		t.Fatalf("GetDataRoot = %q, want data root", svc.GetDataRoot())
	}
	want := filepath.Join(svc.GetAppRoot(), "logs")
	if want != logDir {
		t.Fatalf("log path = %q, want %q", want, logDir)
	}
}

func TestSystemServiceScanJava(t *testing.T) {
	svc := &services.SystemService{}
	installs, err := svc.ScanJava()
	if err != nil {
		t.Fatalf("ScanJava: %v", err)
	}
	t.Logf("Found %d Java installations", len(installs))
}

func TestHomeServiceCancelRequiresActiveOperation(t *testing.T) {
	dir := t.TempDir()
	mgr := instances.NewManager(dir, instances.DefaultLauncherDefaults())
	registry := instances.NewRegistry()
	svc := &services.HomeService{DataRoot: dir, Instances: mgr, Registry: registry}

	inst, err := mgr.Create("Test", "1.21.4", instances.LoaderVanilla)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.CancelInstance(inst.ID); err == nil {
		t.Fatal("expected no-active-operation error")
	}
	if _, err := registry.Start(inst.ID, instances.OpDownload); err != nil {
		t.Fatal(err)
	}
	if err := svc.CancelInstance(inst.ID); err != nil {
		t.Fatalf("CancelInstance: %v", err)
	}
	if registry.IsActive(inst.ID) {
		t.Fatal("operation remains active")
	}
}

func TestErrorCodes(t *testing.T) {
	dir := t.TempDir()
	mgr := instances.NewManager(dir, instances.DefaultLauncherDefaults())
	svc := &services.InstanceService{DataRoot: dir, Manager: mgr}

	// Validation error
	_, err := svc.CreateInstance("", "", "")
	if err == nil {
		t.Fatal("expected error")
	}
	svcErr, ok := err.(*services.ServiceError)
	if !ok {
		t.Fatal("expected ServiceError")
	}
	if svcErr.Code != services.ErrCodeValidation {
		t.Errorf("Code = %q, want %q", svcErr.Code, services.ErrCodeValidation)
	}

	// Not found error
	_, err = svc.GetInstance("nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
	svcErr, ok = err.(*services.ServiceError)
	if !ok {
		t.Fatal("expected ServiceError")
	}
	if svcErr.Code != services.ErrCodeNotFound {
		t.Errorf("Code = %q, want %q", svcErr.Code, services.ErrCodeNotFound)
	}
}

func TestLegacyTokenFieldsStrippedOnLoad(t *testing.T) {
	dir := t.TempDir()
	legacy := `{"schemaVersion":1,"accounts":[{"uuid":"u-1","username":"Steve","type":"microsoft","accessToken":"plain-secret","refreshToken":"plain-refresh","expiresAt":"2026-01-01T00:00:00Z"}]}` + "\n"
	if err := os.WriteFile(filepath.Join(dir, "accounts.json"), []byte(legacy), 0600); err != nil {
		t.Fatal(err)
	}
	svc := &services.AccountService{DataRoot: dir}
	accounts, err := svc.ListAccounts()
	if err != nil {
		t.Fatalf("ListAccounts: %v", err)
	}
	if len(accounts) != 1 || accounts[0].Type != "microsoft" {
		t.Fatalf("accounts = %+v", accounts)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "accounts.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"plain-secret", "plain-refresh", "accessToken", "refreshToken", "expiresAt"} {
		if strings.Contains(string(raw), secret) {
			t.Fatalf("accounts.json still contains %q: %s", secret, raw)
		}
	}
}
