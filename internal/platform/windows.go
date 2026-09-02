//go:build windows

package platform

import (
	"os/exec"
)

// OpenFileManager opens the specified path in Windows Explorer.
func OpenFileManager(path string) error {
	return exec.Command("explorer", path).Start()
}
