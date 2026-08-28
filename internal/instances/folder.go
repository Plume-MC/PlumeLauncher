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
	// Scan all instance directories to find the one matching this ID
	instancesDir := filepath.Join(dataRoot, "instances")
	entries, err := os.ReadDir(instancesDir)
	if err != nil {
		return fmt.Errorf("instance directory not found: %s", instanceID)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		inst, err := readInstanceJSON(filepath.Join(instancesDir, entry.Name(), "instance.json"))
		if err != nil {
			continue
		}
		if inst.ID == instanceID {
			dir := filepath.Join(instancesDir, entry.Name())
			relative, err := filepath.Rel(dataRoot, dir)
			if err != nil {
				return err
			}
			dir, err = security.ResolveUnderRoot(dataRoot, filepath.ToSlash(relative))
			if err != nil {
				return err
			}
			switch runtime.GOOS {
			case "windows":
				return execCmd("explorer", dir)
			case "linux":
				return execCmd("xdg-open", dir)
			default:
				return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
			}
		}
	}
	return fmt.Errorf("instance directory not found: %s", instanceID)
}

func execCmd(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Start()
}
