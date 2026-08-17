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

func TestMissingTokenHasSafeError(t *testing.T) {
	if !errors.Is(auth.ErrTokenNotFound, auth.ErrTokenNotFound) {
		t.Fatal("missing token error must be identifiable without a fallback")
	}
}
