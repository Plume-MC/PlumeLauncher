package instances

import (
	"encoding/json"
	"os"
	"path/filepath"

	"plumelauncher/internal/storage"
)

// ConfigFile represents the launcher config.json structure.
type ConfigFile struct {
	storage.Document
	Settings LauncherDefaults `json:"settings"`
}

// SaveConfig persists launcher settings to config.json.
func SaveConfig(dataRoot string, defaults LauncherDefaults) error {
	config := ConfigFile{
		Document: storage.Document{SchemaVersion: 1},
		Settings: defaults,
	}
	path := filepath.Join(dataRoot, "config.json")
	return storage.WriteJSON(path, config)
}

// LoadConfig loads launcher settings from config.json.
// Returns default settings if file doesn't exist.
func LoadConfig(dataRoot string) (LauncherDefaults, error) {
	path := filepath.Join(dataRoot, "config.json")

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultLauncherDefaults(), nil
		}
		return LauncherDefaults{}, err
	}

	var config ConfigFile
	if err := json.Unmarshal(data, &config); err != nil {
		return LauncherDefaults{}, err
	}

	return config.Settings, nil
}
