package instances

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"plumelauncher/internal/security"
)

// OpenInstanceFolder opens the instance directory in the OS file manager.
// Validates that the path is under DataRoot.
func OpenInstanceFolder(dataRoot string, instanceID string) error {
	dir, err := instanceDirectory(dataRoot, instanceID)
	if err != nil {
		return err
	}
	relative, err := filepath.Rel(dataRoot, dir)
	if err != nil {
		return err
	}
	dir, err = security.ResolveUnderRoot(dataRoot, filepath.ToSlash(relative))
	if err != nil {
		return err
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
