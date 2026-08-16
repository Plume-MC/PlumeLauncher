package bootstrap

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"

	"plumelauncher/internal/storage"
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
		if configured, err := configuredDataRoot(); err != nil {
			return "", err
		} else {
			root = configured
		}
	}
	if root == "" {
		root = defaultDataRoot()
	}

	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", err
	}

	return root, nil
}

type rootConfig struct {
	storage.Document
	DataRoot string `json:"dataRoot"`
}

func configuredDataRoot() (string, error) {
	var config rootConfig
	if err := storage.ReadJSON(filepath.Join(defaultDataRoot(), "bootstrap.json"), &config); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	return config.DataRoot, nil
}

// SetDataRoot stores the root to use on the next app start without moving data.
func SetDataRoot(root string) error {
	if root == "" {
		return os.ErrInvalid
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(absolute, 0o755); err != nil {
		return err
	}
	return storage.WriteJSON(filepath.Join(defaultDataRoot(), "bootstrap.json"), rootConfig{
		Document: storage.Document{SchemaVersion: 1},
		DataRoot: absolute,
	})
}

// ResolveLogDir returns the log directory path and ensures it exists.
func ResolveLogDir(dataRoot string) (string, error) {
	logDir := filepath.Join(dataRoot, "logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return "", err
	}
	return logDir, nil
}
