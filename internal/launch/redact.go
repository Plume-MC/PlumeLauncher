package launch

import "plumelauncher/internal/logging"

// Redact removes session tokens before command output reaches the UI or an error.
func Redact(value string, secrets ...string) string {
	return logging.Redact(value, secrets...)
}
