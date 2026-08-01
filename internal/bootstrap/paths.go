package bootstrap

import (
	"os"
	"path/filepath"
	"runtime"
)

func dataRootEnvName() string {
	return "PLUME_DATA_ROOT"
}

func defaultDataRoot() string {
	if runtime.GOOS == "windows" {
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			localAppData = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local")
		}
		return filepath.Join(localAppData, "PlumeLauncher")
	}

	// Linux/Unix: XDG_DATA_HOME or ~/.local/share
	xdg := os.Getenv("XDG_DATA_HOME")
	if xdg != "" {
		return filepath.Join(xdg, "PlumeLauncher")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "PlumeLauncher")
}

// ResolveDataRoot returns the data root directory.
// If customRoot is non-empty, it is used directly.
// Otherwise, the PLUME_DATA_ROOT env var is checked, then the platform default.
func ResolveDataRoot(customRoot string) (string, error) {
	root := customRoot
	if root == "" {
		root = os.Getenv(dataRootEnvName())
	}
	if root == "" {
		root = defaultDataRoot()
	}

	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", err
	}

	return root, nil
}

// ResolveLogDir returns the log directory path and ensures it exists.
func ResolveLogDir(dataRoot string) (string, error) {
	logDir := filepath.Join(dataRoot, "logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return "", err
	}
	return logDir, nil
}
