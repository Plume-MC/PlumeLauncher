package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Document is embedded in all JSON root objects for version tracking.
type Document struct {
	SchemaVersion int `json:"schemaVersion"`
}

// WriteJSON atomically writes a Go value to a JSON file.
// It writes to a sibling temporary file, syncs, then renames to the target path.
func WriteJSON(path string, v any) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}

	tmpPath := path + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("create tmp: %w", err)
	}

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("encode: %w", err)
	}

	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("sync: %w", err)
	}

	if err := f.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("close: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename: %w", err)
	}

	return nil
}

// ReadJSON reads a JSON file into v and validates that schemaVersion is present.
func ReadJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}

	if len(data) == 0 {
		return fmt.Errorf("empty file")
	}

	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	// Validate schemaVersion is present by checking the raw structure
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("validate: %w", err)
	}
	if _, ok := raw["schemaVersion"]; !ok {
		return fmt.Errorf("missing required field: schemaVersion")
	}

	return nil
}
