package platform

import (
	"fmt"
	"net/url"
	"strings"
)

const browserURLHost = "account.ely.by"

// browserURLHosts are the hosts the launcher may open in the system browser:
// the Ely.by device-code page and the Microsoft (Entra) device-code page.
var browserURLHosts = map[string]bool{
	"account.ely.by":            true,
	"login.microsoftonline.com": true,
	"microsoft.com":             true,
	"login.live.com":            true,
}

func validateBrowserURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("unsupported browser URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("unsupported browser URL")
	}
	if parsed.User != nil || parsed.Port() != "" || !browserURLHosts[strings.ToLower(parsed.Hostname())] {
		return fmt.Errorf("unsupported browser URL")
	}
	return nil
}
