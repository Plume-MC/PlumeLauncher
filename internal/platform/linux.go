//go:build linux

package platform

import (
	"os/exec"
)

// OpenFileManager opens the specified path in the default file manager.
func OpenFileManager(path string) error {
	return exec.Command("xdg-open", path).Start()
}
