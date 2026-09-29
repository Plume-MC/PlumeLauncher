//go:build windows

package platform

import (
	"os"
	"os/exec"
)

// OpenFileManager opens the specified path in Windows Explorer.
func OpenFileManager(path string) error {
	return exec.Command(explorerExecutable(os.Getenv), path).Start()
}

// OpenBrowserURL opens an allowlisted external URL in the default browser.
func OpenBrowserURL(rawURL string) error {
	if err := validateBrowserURL(rawURL); err != nil {
		return err
	}
	return exec.Command(explorerExecutable(os.Getenv), rawURL).Start()
}
