package instances

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// OpenInstanceFolder opens the instance directory in the OS file manager.
// Validates that the path is under DataRoot.
func OpenInstanceFolder(dataRoot string, instanceID string) error {
	dir := filepath.Join(dataRoot, "instances", instanceID)

	// Validate path is under DataRoot
	absDir, _ := filepath.Abs(dir)
	absRoot, _ := filepath.Abs(dataRoot)
	if len(absDir) < len(absRoot) || absDir[:len(absRoot)] != absRoot {
		return fmt.Errorf("path traversal detected")
	}

	// Verify directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("instance directory not found: %s", instanceID)
	}

	// Open in file manager
	switch runtime.GOOS {
	case "windows":
		return execCmd("explorer", dir)
	case "linux":
		return execCmd("xdg-open", dir)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func execCmd(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Start()
}
