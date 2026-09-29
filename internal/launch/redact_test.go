package launch_test

import (
	"strings"
	"testing"

	"plumelauncher/internal/launch"
)

func TestRedactRemovesSessionToken(t *testing.T) {
	got := launch.Redact("token=secret-token", "secret-token")
	if strings.Contains(got, "secret-token") || !strings.Contains(got, "[redacted]") {
		t.Fatalf("Redact = %q", got)
	}
}

func TestRedactRemovesOAuthDeviceSecrets(t *testing.T) {
	// OAuth artifacts must never reach logs: device code, user code,
	// access token, and refresh token are all redacted.
	secrets := []string{"dev-123-secret", "ABCD-1234", "oauth-access-1", "oauth-refresh-1"}
	line := "device dev-123-secret user ABCD-1234 access oauth-access-1 refresh oauth-refresh-1"
	got := launch.Redact(line, secrets...)
	for _, secret := range secrets {
		if strings.Contains(got, secret) {
			t.Fatalf("Redact leaked %q: %q", secret, got)
		}
	}
}
