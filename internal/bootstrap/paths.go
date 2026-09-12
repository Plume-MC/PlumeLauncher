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

// portableMarkerName is the opt-in file that, when placed next to the
// executable, makes the launcher store its data beside the app instead of
// the platform default (portable mode).
func portableMarkerName() string {
	return "portable.txt"
}

// portableDataRoot returns exeDir when portable mode is active, else "".
// It is a pure function of exeDir so tests do not depend on os.Executable.
func portableDataRoot(exeDir string) string {
	if exeDir == "" {
		return ""
	}
	if _, err := os.Stat(filepath.Join(exeDir, portableMarkerName())); err != nil {
		return ""
	}
	return exeDir
}

// executableDir returns the directory holding the running executable,
// or "" when it cannot be determined (then portable mode stays off).
func executableDir() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Dir(exe)
}

// IsPortableMode reports whether dataRoot is the portable folder beside
// the running executable (i.e. portable.txt sits next to the exe and the
// resolved data root is the exe directory).
func IsPortableMode(dataRoot string) bool {
	return isPortableRoot(dataRoot, executableDir())
}

func isPortableRoot(dataRoot, exeDir string) bool {
	return exeDir != "" && dataRoot != "" && dataRoot == exeDir && portableDataRoot(exeDir) != ""
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
// Priority: customRoot arg → PLUME_DATA_ROOT → portable marker next to the
// executable → bootstrap.json → platform default.
func ResolveGameRoot(customRoot string) (string, error) {
	return resolveGameRoot(customRoot, os.Getenv(dataRootEnvName()), executableDir(), configuredGameRoot)
}

func resolveGameRoot(customRoot, envRoot, exeDir string, configured func() (string, error)) (string, error) {
	root := customRoot
	if root == "" {
		root = envRoot
	}
	if root == "" {
		root = portableDataRoot(exeDir)
	}
	if root == "" {
		if configured, err := configured(); err != nil {
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
