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

func defaultAppRoot() string {
	if runtime.GOOS == "windows" {
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			localAppData = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local")
		}
		return filepath.Join(localAppData, "PlumeLauncher")
	}

	xdg := os.Getenv("XDG_DATA_HOME")
	if xdg != "" {
		return filepath.Join(xdg, "PlumeLauncher")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "PlumeLauncher")
}

// ResolveGameRoot returns the canonical launcher data directory.
// Priority: customRoot arg → PLUME_DATA_ROOT → bootstrap.json → platform default.
func ResolveGameRoot(customRoot string) (string, error) {
	root := customRoot
	if root == "" {
		root = os.Getenv(dataRootEnvName())
	}
	if root == "" {
		if configured, err := configuredGameRoot(); err != nil {
			return "", err
		} else {
			root = configured
		}
	}
	if root == "" {
		root = defaultAppRoot()
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

func configuredGameRoot() (string, error) {
	var config rootConfig
	if err := storage.ReadJSON(filepath.Join(defaultAppRoot(), "bootstrap.json"), &config); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	return config.DataRoot, nil
}

// SetDataRoot stores the canonical data root to use on the next app start without moving data.
// bootstrap.json remains in the platform bootstrap directory so the selected root can be discovered.
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
	appRoot := defaultAppRoot()
	if err := os.MkdirAll(appRoot, 0o755); err != nil {
		return err
	}
	return storage.WriteJSON(filepath.Join(appRoot, "bootstrap.json"), rootConfig{
		Document: storage.Document{SchemaVersion: 1},
		DataRoot: absolute,
	})
}
