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
