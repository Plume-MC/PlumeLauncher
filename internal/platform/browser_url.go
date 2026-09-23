package platform

import (
	"fmt"
	"net/url"
	"strings"
)

const browserURLHost = "account.ely.by"

func validateBrowserURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("unsupported browser URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("unsupported browser URL")
	}
	if parsed.User != nil || parsed.Port() != "" || !strings.EqualFold(parsed.Hostname(), browserURLHost) {
		return fmt.Errorf("unsupported browser URL")
	}
	return nil
}
