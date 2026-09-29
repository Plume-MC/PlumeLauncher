//go:build linux

package platform

import (
	"os/exec"
)

// OpenFileManager opens the specified path in the default file manager.
func OpenFileManager(path string) error {
	return exec.Command("xdg-open", path).Start()
}

// OpenBrowserURL opens an allowlisted external URL in the default browser.
func OpenBrowserURL(rawURL string) error {
	if err := validateBrowserURL(rawURL); err != nil {
		return err
	}
	return exec.Command("xdg-open", rawURL).Start()
}
