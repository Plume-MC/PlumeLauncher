package launch

import "strings"

// Redact removes session tokens before command output reaches the UI or an error.
func Redact(value string, secrets ...string) string {
	for _, secret := range secrets {
		if secret != "" {
			value = strings.ReplaceAll(value, secret, "[redacted]")
		}
	}
	return value
}
