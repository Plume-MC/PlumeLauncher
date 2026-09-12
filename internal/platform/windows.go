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
