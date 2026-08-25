package instances_test

import (
	"os"
	"path/filepath"
	"testing"

	"plumelauncher/internal/instances"
)

func TestSaveLoadConfig(t *testing.T) {
	dir := t.TempDir()

	defaults := instances.DefaultLauncherDefaults()
	defaults.DefaultMinRamMB = 2048
	defaults.DefaultMaxRamMB = 8192
	defaults.DefaultJavaPath = "C:/Java/jdk-21/bin/java.exe"
	defaults.CustomJavaPaths = []string{"C:/Java/jdk-8/bin/java.exe"}
	defaults.JavaDefaultInitialized = true

	err := instances.SaveConfig(dir, defaults)
	if err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	loaded, err := instances.LoadConfig(dir)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if loaded.DefaultMinRamMB != 2048 {
		t.Errorf("DefaultMinRamMB = %d, want 2048", loaded.DefaultMinRamMB)
	}
	if loaded.DefaultMaxRamMB != 8192 {
		t.Errorf("DefaultMaxRamMB = %d, want 8192", loaded.DefaultMaxRamMB)
	}
	if loaded.DefaultJavaPath != defaults.DefaultJavaPath || len(loaded.CustomJavaPaths) != 1 || !loaded.JavaDefaultInitialized {
		t.Fatalf("managed Java settings were not preserved: %#v", loaded)
	}
}

func TestLoadConfigDefault(t *testing.T) {
	dir := t.TempDir()

	loaded, err := instances.LoadConfig(dir)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if loaded.Theme != "dark" {
		t.Errorf("Theme = %q, want %q", loaded.Theme, "dark")
	}
	if loaded.DefaultMinRamMB != 1024 {
		t.Errorf("DefaultMinRamMB = %d, want 1024", loaded.DefaultMinRamMB)
	}
}

func TestLoadConfigCorrupted(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	os.WriteFile(path, []byte("not json"), 0o644)

	_, err := instances.LoadConfig(dir)
	if err == nil {
		t.Fatal("expected error for corrupted config")
	}
}

func TestSaveConfigAtomicWrite(t *testing.T) {
	dir := t.TempDir()

	// Save original
	defaults := instances.DefaultLauncherDefaults()
	instances.SaveConfig(dir, defaults)

	// Save updated
	defaults.DefaultMinRamMB = 4096
	err := instances.SaveConfig(dir, defaults)
	if err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	// Load and verify
	loaded, _ := instances.LoadConfig(dir)
	if loaded.DefaultMinRamMB != 4096 {
		t.Errorf("DefaultMinRamMB = %d, want 4096", loaded.DefaultMinRamMB)
	}

	// Verify no temp file left
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".tmp" {
			t.Errorf("temp file left: %s", e.Name())
		}
	}
}
