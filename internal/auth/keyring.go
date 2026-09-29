package auth

import (
	"errors"
	"fmt"

	"github.com/zalando/go-keyring"
)

const keyringService = "PlumeLauncher"

var ErrTokenNotFound = errors.New("session token not found")

// Keyring abstracts OS keyring operations.
type Keyring interface {
	Set(key, value string) error
	Get(key string) (string, error)
	Delete(key string) error
}

// OSKeyring stores tokens only in the operating system keyring.
type OSKeyring struct{}

func (OSKeyring) Set(key, value string) error {
	if err := keyring.Set(keyringService, key, value); err != nil {
		return fmt.Errorf("store session token: %w", err)
	}
	return nil
}

func (OSKeyring) Get(key string) (string, error) {
	value, err := keyring.Get(keyringService, key)
	if errors.Is(err, keyring.ErrNotFound) {
		return "", ErrTokenNotFound
	}
	if err != nil {
		return "", fmt.Errorf("read session token: %w", err)
	}
	return value, nil
}

func (OSKeyring) Delete(key string) error {
	err := keyring.Delete(keyringService, key)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("delete session token: %w", err)
	}
	return nil
}

func SessionKey(accountUUID string) string {
	return "ely.by:" + accountUUID
}

// MicrosoftRefreshKey scopes a Microsoft refresh token to one account.
// Refresh tokens live only in the OS keyring, never in JSON, logs, or bindings.
func MicrosoftRefreshKey(accountUUID string) string {
	return "microsoft-refresh:" + accountUUID
}

// MicrosoftAccessKey scopes a Minecraft access token to one Microsoft account.
// The game access token is short-lived and keyring-only, like refresh tokens.
func MicrosoftAccessKey(accountUUID string) string {
	return "microsoft-access:" + accountUUID
}

// MicrosoftExpiryKey scopes the access-token expiry (RFC3339) to one account.
func MicrosoftExpiryKey(accountUUID string) string {
	return "microsoft-expiry:" + accountUUID
}
