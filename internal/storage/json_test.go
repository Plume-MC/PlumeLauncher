package storage_test

import (
	"os"
	"path/filepath"
	"testing"

	"plumelauncher/internal/storage"
)

type testDoc struct {
	storage.Document
	Name string `json:"name"`
}

func TestWriteReadRoundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")

	original := &testDoc{
		Document: storage.Document{SchemaVersion: 1},
		Name:     "hello",
	}

	if err := storage.WriteJSON(path, original); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}

	var loaded testDoc
	if err := storage.ReadJSON(path, &loaded); err != nil {
		t.Fatalf("ReadJSON: %v", err)
	}

	if loaded.SchemaVersion != 1 {
		t.Errorf("SchemaVersion = %d, want 1", loaded.SchemaVersion)
	}
	if loaded.Name != "hello" {
		t.Errorf("Name = %q, want %q", loaded.Name, "hello")
	}
}

func TestAtomicWriteRetainsPrevious(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")

	// Write original
	original := &testDoc{
		Document: storage.Document{SchemaVersion: 1},
		Name:     "original",
	}
	if err := storage.WriteJSON(path, original); err != nil {
		t.Fatalf("WriteJSON original: %v", err)
	}

	// Simulate interrupted write by writing a temp file and NOT renaming
	tmpPath := path + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString(`{"schemaVersion":1,"name":"interrupted"}`)
	f.Close()
	// Do NOT rename — simulate crash

	// Original should still be intact
	var loaded testDoc
	if err := storage.ReadJSON(path, &loaded); err != nil {
		t.Fatalf("ReadJSON after interrupted write: %v", err)
	}
	if loaded.Name != "original" {
		t.Errorf("Name = %q after interrupted write, want %q", loaded.Name, "original")
	}

	// Cleanup temp file
	os.Remove(tmpPath)
}

func TestReadJSONRequiresSchemaVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "no_schema.json")

	// Write file without schemaVersion
	if err := os.WriteFile(path, []byte(`{"name":"test"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	var loaded testDoc
	err := storage.ReadJSON(path, &loaded)
	if err == nil {
		t.Fatal("expected error for missing schemaVersion")
	}
}

func TestReadJSONRequiresNonEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.json")

	if err := os.WriteFile(path, []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}

	var loaded testDoc
	err := storage.ReadJSON(path, &loaded)
	if err == nil {
		t.Fatal("expected error for empty file")
	}
}
