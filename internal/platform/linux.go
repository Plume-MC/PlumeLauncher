//go:build linux

package platform

import (
	"fmt"
	"os/exec"
)

// OpenFileManager opens the specified path in the default file manager.
func OpenFileManager(path string) error {
	return exec.Command("xdg-open", path).Start()
}

// platformName returns the OS identifier.
func platformName() string {
	return "linux"
}

// ErrNotImplemented is a placeholder for features not yet available.
var ErrNotImplemented = fmt.Errorf("not implemented")
