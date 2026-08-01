//go:build windows

package platform

import (
	"fmt"
	"os/exec"
)

// OpenFileManager opens the specified path in Windows Explorer.
func OpenFileManager(path string) error {
	return exec.Command("explorer", path).Start()
}

// platformName returns the OS identifier.
func platformName() string {
	return "windows"
}

// ErrNotImplemented is a placeholder for features not yet available.
var ErrNotImplemented = fmt.Errorf("not implemented")
