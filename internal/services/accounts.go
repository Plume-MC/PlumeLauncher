package services

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"

	"plumelauncher/internal/auth"
	"plumelauncher/internal/storage"
)

// AccountService manages user accounts.
type AccountService struct {
	DataRoot   string
	HTTPClient *http.Client
	ElyByURL   string
	Keyring    auth.Keyring
}

func (s *AccountService) LoginElyBy(username, password string) (*Account, error) {
	if username == "" || password == "" {
		return nil, NewValidationError("username and password are required", "username", "password")
	}
	session, err := (auth.ElyByClient{HTTPClient: s.HTTPClient, BaseURL: s.ElyByURL}).Authenticate(username, password)
	if err != nil {
		return nil, NewUpstreamError("Ely.by authentication failed")
	}
	account := Account{UUID: session.SelectedProfile.ID, Username: session.SelectedProfile.Name, DisplayName: session.SelectedProfile.Name, Type: "ely.by", Selected: true}
	key := auth.SessionKey(account.UUID)
	store := s.tokenStore()
	if err := store.Set(key, session.AccessToken); err != nil {
		return nil, NewUpstreamError("secure session storage is unavailable; please retry after fixing your OS keyring")
	}
	keepToken := false
	defer func() {
		if !keepToken {
			_ = store.Delete(key)
		}
	}()
	accounts, err := s.loadAccounts()
	if err != nil {
		return nil, NewInternalError("failed to load accounts")
	}
	for i, existing := range accounts {
		if existing.UUID == account.UUID {
			for j := range accounts {
				accounts[j].Selected = false
			}
			accounts[i] = account
			if err := s.saveAccounts(accounts); err != nil {
				return nil, NewInternalError("failed to save account")
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
		return nil, NewInternalError("failed to save account")
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
	session, err := (auth.ElyByClient{HTTPClient: s.HTTPClient, BaseURL: s.ElyByURL}).Refresh(token)
	if err != nil {
		return nil, NewUpstreamError("Ely.by session refresh failed; please sign in again")
	}
	if err := s.tokenStore().Set(auth.SessionKey(account.UUID), session.AccessToken); err != nil {
		return nil, NewUpstreamError("secure session storage is unavailable")
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
		return NewUpstreamError("secure session storage is unavailable")
	}
	accounts, err := s.loadAccounts()
	if err != nil {
		return NewInternalError("failed to load accounts")
	}
	for i := range accounts {
		if accounts[i].UUID == accountUUID {
			accounts = append(accounts[:i], accounts[i+1:]...)
			break
		}
	}
	if err := s.saveAccounts(accounts); err != nil {
		return NewInternalError("failed to save account")
	}
	return nil
}

// DeleteAccount removes an offline profile or revokes and removes an Ely.by session.
func (s *AccountService) DeleteAccount(accountUUID string) error {
	accounts, err := s.loadAccounts()
	if err != nil {
		return NewInternalError("failed to load accounts")
	}
	for _, account := range accounts {
		if account.UUID != accountUUID {
			continue
		}
		if account.Type == "ely.by" {
			return s.LogoutElyBy(accountUUID)
		}
		for i := range accounts {
			if accounts[i].UUID == accountUUID {
				accounts = append(accounts[:i], accounts[i+1:]...)
				break
			}
		}
		if err := s.saveAccounts(accounts); err != nil {
			return NewInternalError("failed to save account")
		}
		return nil
	}
	return NewNotFoundError("account not found")
}

func (s *AccountService) elyByAccount(uuid string) (Account, error) {
	accounts, err := s.loadAccounts()
	if err != nil {
		return Account{}, NewInternalError("failed to load accounts")
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
		return NewInternalError("failed to load accounts")
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
		return NewInternalError("failed to save account")
	}
	return nil
}

func (s *AccountService) selectedAccount() (Account, string, error) {
	accounts, err := s.loadAccounts()
	if err != nil {
		return Account{}, "", NewInternalError("failed to load accounts")
	}
	for _, account := range accounts {
		if !account.Selected {
			continue
		}
		if account.Type != "ely.by" {
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
		return nil, NewInternalError("failed to load accounts")
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
		return nil, NewInternalError("failed to save account")
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
