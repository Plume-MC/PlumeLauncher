package services

import (
	"context"
	"errors"
	"time"

	"plumelauncher/internal/auth"
)

// elyOAuthTimeout bounds the whole Finish poll loop so the frontend
// promise always resolves even if the player never confirms on the
// Ely.by website.
const elyOAuthTimeout = 10 * time.Minute

// elyOAuthClient builds the device-code client with service overrides.
func (s *AccountService) elyOAuthClient() auth.ElyOAuthClient {
	return auth.ElyOAuthClient{
		HTTPClient: s.HTTPClient,
		BaseURL:    s.ElyOAuthBaseURL,
		ClientID:   s.ElyOAuthClientID,
	}
}

// StartElyByOAuth opens an Ely.by device-code session. The frontend shows
// the returned user code + verification URI; the player confirms on the
// Ely.by website. No password ever enters the launcher.
func (s *AccountService) StartElyByOAuth() (auth.ElyDeviceCodeStart, error) {
	start, err := s.elyOAuthClient().StartDeviceCode(context.Background())
	if err != nil {
		return auth.ElyDeviceCodeStart{}, elyOAuthServiceError(err)
	}
	return start, nil
}

// FinishElyByOAuth polls once for the approved device code and creates
// the Ely.by session. The frontend polls this method at the interval
// from StartElyByOAuth until success, denial, expiry, or cancel.
//
// Tokens live only in the OS keyring; accounts.json stays metadata-only.
// On any failure after the token is stored, the token is removed so no
// orphan secret survives.
func (s *AccountService) FinishElyByOAuth(deviceCode string) (*Account, error) {
	if deviceCode == "" {
		return nil, NewValidationError("Enter the Ely.by code first.", "deviceCode")
	}
	accessToken, err := s.elyOAuthClient().PollDeviceToken(context.Background(), deviceCode)
	if err != nil {
		return nil, elyOAuthServiceError(err)
	}
	info, err := s.elyOAuthClient().FetchAccountInfo(context.Background(), accessToken)
	if err != nil {
		return nil, elyOAuthServiceError(err)
	}
	account := Account{
		UUID:        info.UUID,
		Username:    info.Username,
		DisplayName: info.Username,
		Type:        AccountTypeElyBy,
		Selected:    true,
		OAuth:       true,
	}
	key := auth.SessionKey(account.UUID)
	store := s.tokenStore()
	if err := store.Set(key, accessToken); err != nil {
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
	for i := range accounts {
		accounts[i].Selected = false
		if accounts[i].UUID == account.UUID && accounts[i].Type == AccountTypeElyBy {
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

// CancelElyByOAuth ends a local device-code wait. The session itself
// lives on the Ely.by website, so there is nothing to revoke locally;
// the method exists so the frontend cancel path is explicit and typed.
func (s *AccountService) CancelElyByOAuth(_ string) error {
	return nil
}

// elyOAuthServiceError maps device-code failures to user-facing errors
// without leaking codes, tokens, or secrets.
func elyOAuthServiceError(err error) *ServiceError {
	if errors.Is(err, auth.ErrOAuthPending) {
		return NewUpstreamError("Waiting for Ely.by approval. Confirm the code on the Ely.by website.")
	}
	if errors.Is(err, auth.ErrOAuthExpired) {
		return NewUpstreamError("The Ely.by code expired. Start sign-in again for a fresh code.")
	}
	if errors.Is(err, auth.ErrOAuthDenied) {
		return NewUpstreamError("Ely.by sign-in was denied. Try again if this was a mistake.")
	}
	var typed *auth.ElyOAuthError
	if errors.As(err, &typed) {
		switch typed.Code {
		case "invalid_client":
			return NewUpstreamError("Ely.by sign-in is not configured yet. The launcher maintainer must register the application.")
		case "invalid_scope", "invalid_request", "invalid_response":
			return NewInternalError("Ely.by sign-in returned an unexpected response. Try again.")
		}
		return NewUpstreamError("Unable to sign in to Ely.by. Check your connection and try again.")
	}
	return NewUpstreamError("Unable to sign in to Ely.by. Check your connection and try again.")
}
