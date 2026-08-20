package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const CurrentSchemaVersion = 1

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

	f, err := os.CreateTemp(dir, "."+filepath.Base(path)+"-*.tmp")
	if err != nil {
		return fmt.Errorf("create tmp: %w", err)
	}
	tmpPath := f.Name()
	defer os.Remove(tmpPath)

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		f.Close()
		return fmt.Errorf("encode: %w", err)
	}

	if err := f.Sync(); err != nil {
		f.Close()
		return fmt.Errorf("sync: %w", err)
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("close: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename: %w", err)
	}
	if err := syncDirectory(dir); err != nil {
		return fmt.Errorf("sync dir: %w", err)
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
	rawVersion, ok := raw["schemaVersion"]
	if !ok {
		return fmt.Errorf("missing required field: schemaVersion")
	}
	var schemaVersion int
	if err := json.Unmarshal(rawVersion, &schemaVersion); err != nil || schemaVersion < 1 || schemaVersion > CurrentSchemaVersion {
		return fmt.Errorf("unsupported schemaVersion")
	}

	return nil
}

func syncDirectory(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	err = dir.Sync()
	closeErr := dir.Close()
	if err == nil {
		err = closeErr
	}
	if runtime.GOOS == "windows" && err != nil {
		return nil
	}
	return err
}
