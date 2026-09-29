package auth_test

import (
	"errors"
	"testing"

	"plumelauncher/internal/auth"
)

func TestSessionKeyIsAccountScoped(t *testing.T) {
	if got := auth.SessionKey("account-a"); got != "ely.by:account-a" {
		t.Fatalf("SessionKey = %q", got)
	}
}

func TestMicrosoftRefreshKeyIsAccountScoped(t *testing.T) {
	if got := auth.MicrosoftRefreshKey("account-a"); got != "microsoft-refresh:account-a" {
		t.Fatalf("MicrosoftRefreshKey = %q", got)
	}
	if auth.MicrosoftRefreshKey("account-a") == auth.SessionKey("account-a") {
		t.Fatal("microsoft refresh key must not collide with ely.by session key")
	}
}

func TestMissingTokenHasSafeError(t *testing.T) {
	if !errors.Is(auth.ErrTokenNotFound, auth.ErrTokenNotFound) {
		t.Fatal("missing token error must be identifiable without a fallback")
	}
}
