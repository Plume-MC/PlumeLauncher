package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"plumelauncher/internal/auth"
	"plumelauncher/internal/storage"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// AccountService manages user accounts.
type AccountService struct {
	DataRoot   string
	HTTPClient *http.Client
	ElyByURL   string
	Keyring    auth.Keyring
	App        *application.App
	// ElyOAuthBaseURL overrides the Ely.by OAuth host (tests only).
	ElyOAuthBaseURL string
	// ElyOAuthClientID is the OAuth application ID registered at
	// https://account.ely.by/dev/. Empty means the built-in placeholder,
	// which the maintainer must replace before release.
	ElyOAuthClientID string
}

func (s *AccountService) LoginElyBy(username, password string) (*Account, error) {
	if username == "" || password == "" {
		return nil, NewValidationError("Enter a username and password.", "username", "password")
	}
	session, err := (auth.ElyByClient{HTTPClient: s.HTTPClient, BaseURL: s.ElyByURL}).Authenticate(username, password)
	if err != nil {
		return nil, NewUpstreamError("Unable to sign in to Ely.by. Check your details and connection, then try again.")
	}
	account := Account{UUID: session.SelectedProfile.ID, Username: session.SelectedProfile.Name, DisplayName: session.SelectedProfile.Name, Type: "ely.by", Selected: true}
	key := auth.SessionKey(account.UUID)
	store := s.tokenStore()
	if err := store.Set(key, session.AccessToken); err != nil {
		return nil, NewUpstreamError("Unable to access secure storage on this device. Try again.")
	}
	keepToken := false
	defer func() {
		if !keepToken {
			_ = store.Delete(key)
		}
	}()
	accounts, err := s.loadAccounts()
	if err != nil {
		return nil, NewInternalError("Unable to load accounts. Restart the launcher and try again.")
	}
	for i, existing := range accounts {
		if existing.UUID == account.UUID {
			for j := range accounts {
				accounts[j].Selected = false
			}
			accounts[i] = account
			if err := s.saveAccounts(accounts); err != nil {
				return nil, NewInternalError("Unable to save the account.")
			}
			keepToken = true
			return &account, nil
		}
	}
	for i := range accounts {
		accounts[i].Selected = false
	}
	accounts = append(accounts, account)
	if err := s.saveAccounts(accounts); err != nil {
		return nil, NewInternalError("Unable to save the account.")
	}
	keepToken = true
	return &account, nil
}

func (s *AccountService) RefreshElyBy(accountUUID string) (*Account, error) {
	account, err := s.elyByAccount(accountUUID)
	if err != nil {
		return nil, err
	}
	token, err := s.tokenStore().Get(auth.SessionKey(account.UUID))
	if err != nil {
		return nil, NewUpstreamError("Ely.by session expired; please sign in again")
	}
	if account.OAuth {
		// Device-code sessions carry no Yggdrasil refresh path. Validate
		// the stored token instead; an invalid token asks for re-login.
		if _, err := s.elyOAuthClient().FetchAccountInfo(context.Background(), token); err != nil {
			return nil, NewUpstreamError("Ely.by session expired; please sign in again")
		}
		return &account, nil
	}
	session, err := (auth.ElyByClient{HTTPClient: s.HTTPClient, BaseURL: s.ElyByURL}).Refresh(token)
	if err != nil {
		return nil, NewUpstreamError("Ely.by session refresh failed; please sign in again")
	}
	if err := s.tokenStore().Set(auth.SessionKey(account.UUID), session.AccessToken); err != nil {
		return nil, NewUpstreamError("Unable to access secure storage on this device.")
	}
	return &account, nil
}

func (s *AccountService) LogoutElyBy(accountUUID string) error {
	account, err := s.elyByAccount(accountUUID)
	if err != nil {
		return err
	}
	token, tokenErr := s.tokenStore().Get(auth.SessionKey(account.UUID))
	if tokenErr == nil {
		_ = (auth.ElyByClient{HTTPClient: s.HTTPClient, BaseURL: s.ElyByURL}).Invalidate(token)
	}
	if err := s.tokenStore().Delete(auth.SessionKey(account.UUID)); err != nil {
		return NewUpstreamError("Unable to access secure storage on this device.")
	}
	accounts, err := s.loadAccounts()
	if err != nil {
		return NewInternalError("Unable to load accounts. Restart the launcher and try again.")
	}
	for i := range accounts {
		if accounts[i].UUID == accountUUID {
			accounts = append(accounts[:i], accounts[i+1:]...)
			break
		}
	}
	if err := s.saveAccounts(accounts); err != nil {
		return NewInternalError("Unable to save the account.")
	}
	return nil
}

// DeleteAccount removes an offline profile or revokes and removes an Ely.by/Microsoft session.
// The last remaining account cannot be removed; add another account first.
func (s *AccountService) DeleteAccount(accountUUID string) error {
	accounts, err := s.loadAccounts()
	if err != nil {
		return NewInternalError("Unable to load accounts. Restart the launcher and try again.")
	}
	for _, account := range accounts {
		if account.UUID != accountUUID {
			continue
		}
		if len(accounts) == 1 {
			return NewConflictError("Cannot remove the last account. Add another account first.")
		}
		if account.Type == AccountTypeElyBy {
			return s.LogoutElyBy(accountUUID)
		}
		if account.Type == AccountTypeMicrosoft {
			return s.LogoutMicrosoft(accountUUID)
		}
		for i := range accounts {
			if accounts[i].UUID == accountUUID {
				accounts = append(accounts[:i], accounts[i+1:]...)
				break
			}
		}
		if err := s.saveAccounts(accounts); err != nil {
			return NewInternalError("Unable to save the account.")
		}
		return nil
	}
	return NewNotFoundError("account not found")
}

// msAuthBootstrapHTML is a static loading page shown before the Microsoft
// login page loads. Wails injects its own cross-platform runtime shim, so no
// manual bridge script is needed here; Microsoft pages never send
// wails:runtime:ready themselves.
const msAuthBootstrapHTML = `<!DOCTYPE html>
<html>
<head><meta charset="utf-8"><title>Microsoft account</title></head>
<body style="margin:0;background:#121011;color:#f5f2f2;font:14px system-ui;display:flex;align-items:center;justify-content:center;height:100vh">
<p>Opening Microsoft login…</p>
</body>
</html>`

// parseMicrosoftCallback extracts the OAuth code from the loopback callback
// request. Only the first code wins; error responses are surfaced so the
// login flow can fail fast instead of hanging until timeout.
func parseMicrosoftCallback(r *http.Request) (string, error) {
	code := r.URL.Query().Get("code")
	if code == "" {
		message := r.URL.Query().Get("error_description")
		if message == "" {
			message = r.URL.Query().Get("error")
		}
		if message == "" {
			message = "no code in callback"
		}
		return "", fmt.Errorf("%s", message)
	}
	return code, nil
}

// LoginMicrosoft runs the full Microsoft login flow in an in-app window.
func (s *AccountService) LoginMicrosoft() (*Account, error) {
	if s.App == nil {
		return nil, NewInternalError("application service is unavailable")
	}

	flow, err := auth.LoginBegin()
	if err != nil {
		return nil, NewUpstreamError("Unable to start Microsoft sign-in. Try again.")
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, NewInternalError("failed to start local server")
	}
	port := listener.Addr().(*net.TCPAddr).Port
	callbackURL := fmt.Sprintf("http://127.0.0.1:%d/callback", port)

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)
	done := make(chan struct{})

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/callback" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		code, callbackErr := parseMicrosoftCallback(r)
		if callbackErr != nil {
			http.Error(w, callbackErr.Error(), http.StatusBadRequest)
			select {
			case errCh <- callbackErr:
			default:
			}
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<html><body style="font:14px system-ui;background:#121011;color:#f5f2f2;display:flex;align-items:center;justify-content:center;height:100vh;margin:0"><p>Login successful. You can close this window.</p></body></html>`)
		select {
		case codeCh <- code:
		default:
		}
	})

	server := &http.Server{Handler: mux}
	go func() { _ = server.Serve(listener) }()
	defer func() {
		_ = server.Shutdown(context.Background())
	}()

	authWindow := s.App.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:                 "microsoft-auth",
		Title:                "Microsoft account",
		Width:                900,
		Height:               700,
		HTML:                 msAuthBootstrapHTML,
		AllowSimpleEventEmit: true,
	})
	defer authWindow.Close()

	// Closing the auth window cancels the flow so the frontend promise
	// resolves immediately instead of hanging until the timeout.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	authWindow.RegisterHook(events.Common.WindowClosing, func(*application.WindowEvent) {
		cancel()
	})

	// Navigate only after the Wails runtime is ready. ExecJS calls made
	// before that are queued, and navigating early risks losing the
	// callback polling script when the page changes to Microsoft.
	var navigateOnce sync.Once
	authWindow.OnWindowEvent(events.Common.WindowRuntimeReady, func(*application.WindowEvent) {
		navigateOnce.Do(func() {
			authWindow.SetURL(flow.AuthRequestURI)
		})
	})

	// Poll for the desktop OAuth callback page and forward the code to loopback.
	callbackScript := fmt.Sprintf(`(() => {
		try {
			const url = String(window.location.href || "");
			if (!url.startsWith("https://login.live.com/oauth20_desktop.srf")) return;
			if (window.__plumeMsAuthHandled) return;
			window.__plumeMsAuthHandled = true;
			const params = new URL(url).searchParams;
			const code = params.get("code");
			const error = params.get("error_description") || params.get("error");
			if (code) {
				window.location.replace(%q + "?code=" + encodeURIComponent(code));
			} else if (error) {
				window.location.replace(%q + "?error=" + encodeURIComponent(error));
			}
		} catch (e) {}
	})();`, callbackURL, callbackURL)

	go func() {
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				authWindow.ExecJS(callbackScript)
			}
		}
	}()
	defer close(done)

	var code string
	select {
	case code = <-codeCh:
	case <-errCh:
		return nil, NewUpstreamError("Microsoft sign-in failed. Try again.")
	case <-ctx.Done():
		return nil, NewCancelledError("Microsoft login cancelled")
	case <-time.After(3 * time.Minute):
		return nil, NewUpstreamError("Microsoft sign-in timed out. Try again.")
	}

	return s.completeMicrosoftLogin(code, flow)
}

// completeMicrosoftLogin finishes the OAuth code exchange and persists the
// metadata-only account row. Tokens stay in the OS keyring.
func (s *AccountService) completeMicrosoftLogin(code string, flow *auth.MicrosoftLoginFlow) (*Account, error) {
	creds, err := auth.LoginFinish(code, flow, s.tokenStore())
	if err != nil {
		return nil, NewUpstreamError("Microsoft sign-in failed. Try again.")
	}
	return s.upsertMicrosoftAccount(creds)
}

// upsertMicrosoftAccount stores session secrets in the keyring and the
// metadata-only row in accounts.json. A persist failure cleans the keyring
// entry so no orphan token remains.
func (s *AccountService) upsertMicrosoftAccount(creds *auth.MicrosoftCredentials) (*Account, error) {
	store := s.tokenStore()
	refreshKey := auth.MicrosoftRefreshKey(creds.UUID)
	if err := store.Set(refreshKey, creds.RefreshToken); err != nil {
		return nil, NewUpstreamError("Unable to access secure storage on this device.")
	}
	keepToken := false
	defer func() {
		if !keepToken {
			_ = store.Delete(refreshKey)
			_ = store.Delete(auth.MicrosoftAccessKey(creds.UUID))
			_ = store.Delete(auth.MicrosoftExpiryKey(creds.UUID))
		}
	}()
	if err := store.Set(auth.MicrosoftAccessKey(creds.UUID), creds.AccessToken); err != nil {
		return nil, NewUpstreamError("Unable to access secure storage on this device.")
	}
	if err := store.Set(auth.MicrosoftExpiryKey(creds.UUID), creds.ExpiresAt.UTC().Format(time.RFC3339)); err != nil {
		return nil, NewUpstreamError("Unable to access secure storage on this device.")
	}

	accounts, err := s.loadAccounts()
	if err != nil {
		return nil, NewInternalError("Unable to load accounts. Restart the launcher and try again.")
	}
	account := Account{
		UUID:        creds.UUID,
		Username:    creds.Username,
		Type:        AccountTypeMicrosoft,
		DisplayName: creds.Username,
		Selected:    true,
	}
	for i := range accounts {
		accounts[i].Selected = false
		if accounts[i].UUID == creds.UUID && accounts[i].Type == AccountTypeMicrosoft {
			accounts[i] = account
			if err := s.saveAccounts(accounts); err != nil {
				return nil, NewInternalError("Unable to save the account.")
			}
			keepToken = true
			return &account, nil
		}
	}
	accounts = append(accounts, account)
	if err := s.saveAccounts(accounts); err != nil {
		return nil, NewInternalError("Unable to save the account.")
	}
	keepToken = true
	return &account, nil
}

// refreshMicrosoftError classifies a refresh failure: a revoked/expired
// session asks for re-login, anything else is transient and retryable.
func refreshMicrosoftError(err error) *ServiceError {
	if errors.Is(err, auth.ErrInvalidGrant) {
		return NewUpstreamError("Microsoft session expired; please sign in again")
	}
	return NewUpstreamError("Microsoft token refresh failed. Check your connection and try again.")
}

// RefreshMicrosoftToken explicitly refreshes a Microsoft session on demand.
func (s *AccountService) RefreshMicrosoftToken(accountUUID string) (*Account, error) {
	account, err := s.microsoftAccount(accountUUID)
	if err != nil {
		return nil, err
	}
	refreshToken, err := s.tokenStore().Get(auth.MicrosoftRefreshKey(account.UUID))
	if err != nil {
		return nil, NewUpstreamError("Microsoft session expired; please sign in again")
	}
	creds, err := auth.RefreshMicrosoft(refreshToken, s.tokenStore())
	if err != nil {
		return nil, refreshMicrosoftError(err)
	}
	account.Username = creds.Username
	account.DisplayName = creds.Username
	if err := s.storeMicrosoftSession(account.UUID, creds); err != nil {
		return nil, err
	}
	accounts, err := s.loadAccounts()
	if err != nil {
		return nil, NewInternalError("Unable to load accounts. Restart the launcher and try again.")
	}
	for i := range accounts {
		if accounts[i].UUID == account.UUID && accounts[i].Type == AccountTypeMicrosoft {
			accounts[i].Username = account.Username
			accounts[i].DisplayName = account.DisplayName
			break
		}
	}
	if err := s.saveAccounts(accounts); err != nil {
		return nil, NewInternalError("Unable to save the account.")
	}
	return &account, nil
}

// LogoutMicrosoft removes the keyring session and the account row.
func (s *AccountService) LogoutMicrosoft(accountUUID string) error {
	if _, err := s.microsoftAccount(accountUUID); err != nil {
		return err
	}
	_ = s.tokenStore().Delete(auth.MicrosoftRefreshKey(accountUUID))
	_ = s.tokenStore().Delete(auth.MicrosoftAccessKey(accountUUID))
	_ = s.tokenStore().Delete(auth.MicrosoftExpiryKey(accountUUID))
	accounts, err := s.loadAccounts()
	if err != nil {
		return NewInternalError("Unable to load accounts. Restart the launcher and try again.")
	}
	for i := range accounts {
		if accounts[i].UUID == accountUUID && accounts[i].Type == AccountTypeMicrosoft {
			accounts = append(accounts[:i], accounts[i+1:]...)
			break
		}
	}
	return s.saveAccounts(accounts)
}

// microsoftLaunchToken resolves the game access token for the selected
// Microsoft account, refreshing first when the stored expiry is near.
func (s *AccountService) microsoftLaunchToken(account Account) (Account, string, error) {
	store := s.tokenStore()
	refreshToken, err := store.Get(auth.MicrosoftRefreshKey(account.UUID))
	if err != nil {
		return Account{}, "", NewUpstreamError("Microsoft session expired; please sign in again")
	}
	var expiresAt time.Time
	if raw, err := store.Get(auth.MicrosoftExpiryKey(account.UUID)); err == nil {
		expiresAt, _ = time.Parse(time.RFC3339, raw)
	}
	access, _, newExpiry, err := auth.EnsureValidMicrosoftToken(refreshToken, expiresAt, store)
	if err != nil {
		return Account{}, "", refreshMicrosoftError(err)
	}
	if access == "" {
		access, err = store.Get(auth.MicrosoftAccessKey(account.UUID))
		if err != nil {
			return Account{}, "", NewUpstreamError("Microsoft session expired; please sign in again")
		}
		return account, access, nil
	}
	if err := s.storeMicrosoftSession(account.UUID, &auth.MicrosoftCredentials{
		AccessToken: access,
		ExpiresAt:   newExpiry,
	}); err != nil {
		return Account{}, "", err
	}
	return account, access, nil
}

// storeMicrosoftSession persists the short-lived access token and its expiry
// in the keyring. The refresh token is written by auth.LoginFinish and
// auth.RefreshMicrosoft themselves.
func (s *AccountService) storeMicrosoftSession(accountUUID string, creds *auth.MicrosoftCredentials) error {
	store := s.tokenStore()
	if err := store.Set(auth.MicrosoftAccessKey(accountUUID), creds.AccessToken); err != nil {
		return NewUpstreamError("Unable to access secure storage on this device.")
	}
	if err := store.Set(auth.MicrosoftExpiryKey(accountUUID), creds.ExpiresAt.UTC().Format(time.RFC3339)); err != nil {
		return NewUpstreamError("Unable to access secure storage on this device.")
	}
	return nil
}

func (s *AccountService) microsoftAccount(uuid string) (Account, error) {
	accounts, err := s.loadAccounts()
	if err != nil {
		return Account{}, NewInternalError("Unable to load accounts. Restart the launcher and try again.")
	}
	for _, account := range accounts {
		if account.UUID == uuid && account.Type == AccountTypeMicrosoft {
			return account, nil
		}
	}
	return Account{}, NewNotFoundError("Microsoft account not found")
}

func (s *AccountService) elyByAccount(uuid string) (Account, error) {
	accounts, err := s.loadAccounts()
	if err != nil {
		return Account{}, NewInternalError("Unable to load accounts. Restart the launcher and try again.")
	}
	for _, account := range accounts {
		if account.UUID == uuid && account.Type == "ely.by" {
			return account, nil
		}
	}
	return Account{}, NewNotFoundError("Ely.by account not found")
}

func (s *AccountService) tokenStore() auth.Keyring {
	if s.Keyring != nil {
		return s.Keyring
	}
	return auth.OSKeyring{}
}

func (s *AccountService) SelectAccount(accountUUID string) error {
	accounts, err := s.loadAccounts()
	if err != nil {
		return NewInternalError("Unable to load accounts. Restart the launcher and try again.")
	}
	found := false
	for i := range accounts {
		accounts[i].Selected = accounts[i].UUID == accountUUID
		found = found || accounts[i].Selected
	}
	if !found {
		return NewNotFoundError("account not found")
	}
	if err := s.saveAccounts(accounts); err != nil {
		return NewInternalError("Unable to save the account.")
	}
	return nil
}

func (s *AccountService) selectedAccount() (Account, string, error) {
	accounts, err := s.loadAccounts()
	if err != nil {
		return Account{}, "", NewInternalError("Unable to load accounts. Restart the launcher and try again.")
	}
	for _, account := range accounts {
		if !account.Selected {
			continue
		}
		if account.Type == AccountTypeMicrosoft {
			return s.microsoftLaunchToken(account)
		}
		if account.Type != AccountTypeElyBy {
			return account, "0", nil
		}
		token, err := s.tokenStore().Get(auth.SessionKey(account.UUID))
		if err != nil {
			return Account{}, "", NewUpstreamError("Ely.by session expired; please sign in again")
		}
		return account, token, nil
	}
	return Account{}, "", NewNotFoundError("no account selected")
}

// Account represents a user account. Accounts.json stores metadata only:
// uuid, username, type, display name and selection. Tokens, passwords and
// OAuth codes must never be added here; they live in the OS keyring.
type Account struct {
	UUID        string `json:"uuid"`
	Username    string `json:"username"`
	Type        string `json:"type"`
	DisplayName string `json:"displayName,omitempty"`
	Selected    bool   `json:"selected,omitempty"`
	// OAuth marks Ely.by sessions created via the device-code flow.
	// OAuth sessions cannot use the Yggdrasil refresh endpoint, so
	// RefreshElyBy asks for a fresh browser sign-in instead.
	OAuth bool `json:"oauth,omitempty"`
}

const (
	AccountTypeOffline   = "offline"
	AccountTypeElyBy     = "ely.by"
	AccountTypeMicrosoft = "microsoft"
)

// legacySecretFields are token-bearing keys from older account records.
// They are stripped on load so plaintext secrets never survive in accounts.json.
var legacySecretFields = []string{"accessToken", "refreshToken", "access_token", "refresh_token", "expiresAt", "expires_at"}

// accountsFile is the JSON structure for accounts.json.
type accountsFile struct {
	storage.Document
	Accounts []Account `json:"accounts"`
}

// CreateOffline creates an offline account with deterministic UUID.
func (s *AccountService) CreateOffline(username string) (*Account, error) {
	if username == "" {
		return nil, NewValidationError("username is required", "username")
	}

	username = auth.NormalizeUsername(username)
	if username == "" {
		return nil, NewValidationError("username is invalid after normalization", "username")
	}

	acc := Account{
		UUID:        auth.OfflineUUID(username),
		Username:    username,
		Type:        "offline",
		DisplayName: auth.OfflineDisplayName(username),
	}

	// Load existing accounts
	accounts, err := s.loadAccounts()
	if err != nil {
		return nil, NewInternalError("Unable to load accounts. Restart the launcher and try again.")
	}
	if len(accounts) == 0 {
		acc.Selected = true
	}

	// Check for duplicate
	for _, a := range accounts {
		if a.UUID == acc.UUID {
			return nil, NewConflictError("account already exists")
		}
	}

	accounts = append(accounts, acc)
	if err := s.saveAccounts(accounts); err != nil {
		return nil, NewInternalError("Unable to save the account.")
	}

	return &acc, nil
}

// ListAccounts returns all accounts.
func (s *AccountService) ListAccounts() ([]Account, error) {
	return s.loadAccounts()
}

func (s *AccountService) loadAccounts() ([]Account, error) {
	path := filepath.Join(s.DataRoot, "accounts.json")
	// Best-effort: purge legacy plaintext secrets before reading.
	_ = stripLegacySecrets(path)
	var file accountsFile
	if err := storage.ReadJSON(path, &file); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	return file.Accounts, nil
}

// stripLegacySecrets removes token-bearing keys from stored account objects.
// It returns true when the file was rewritten. Failures are reported so the
// caller can decide; loadAccounts treats them as non-fatal.
func stripLegacySecrets(path string) bool {
	raw, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var doc struct {
		Accounts []map[string]interface{} `json:"accounts"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return false
	}
	changed := false
	for _, acc := range doc.Accounts {
		for _, field := range legacySecretFields {
			if _, ok := acc[field]; ok {
				delete(acc, field)
				changed = true
			}
		}
	}
	if !changed {
		return false
	}
	var full map[string]interface{}
	if err := json.Unmarshal(raw, &full); err != nil {
		return false
	}
	accounts := make([]interface{}, 0, len(doc.Accounts))
	for _, acc := range doc.Accounts {
		accounts = append(accounts, acc)
	}
	full["accounts"] = accounts
	cleaned, err := json.MarshalIndent(full, "", "  ")
	if err != nil {
		return false
	}
	if err := os.WriteFile(path, append(cleaned, '\n'), 0600); err != nil {
		return false
	}
	return true
}

func (s *AccountService) saveAccounts(accounts []Account) error {
	path := filepath.Join(s.DataRoot, "accounts.json")
	file := accountsFile{
		Document: storage.Document{SchemaVersion: 1},
		Accounts: accounts,
	}
	return storage.WriteJSON(path, file)
}
