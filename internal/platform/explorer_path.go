package platform

import (
	"path/filepath"
)

// explorerExecutable resolves Windows Explorer without relying on %PATH%.
// exec.LookPath fails when the process PATH does not include System32, so
// prefer the absolute path derived from %SystemRoot%.
func explorerExecutable(getenv func(string) string) string {
	if root := getenv("SystemRoot"); root != "" {
		return filepath.Join(root, "explorer.exe")
	}
	return filepath.Join(`C:\Windows`, "explorer.exe")
}
