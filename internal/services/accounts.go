package services

import (
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
	if err := s.tokenStore().Set(auth.SessionKey(account.UUID), session.AccessToken); err != nil {
		return nil, NewUpstreamError("secure session storage is unavailable; please retry after fixing your OS keyring")
	}
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

// Account represents a user account.
type Account struct {
	UUID        string `json:"uuid"`
	Username    string `json:"username"`
	Type        string `json:"type"`
	DisplayName string `json:"displayName,omitempty"`
	Selected    bool   `json:"selected,omitempty"`
}

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
	accounts, _ := s.loadAccounts()
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
	var file accountsFile
	if err := storage.ReadJSON(path, &file); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	return file.Accounts, nil
}

func (s *AccountService) saveAccounts(accounts []Account) error {
	path := filepath.Join(s.DataRoot, "accounts.json")
	file := accountsFile{
		Document: storage.Document{SchemaVersion: 1},
		Accounts: accounts,
	}
	return storage.WriteJSON(path, file)
}
