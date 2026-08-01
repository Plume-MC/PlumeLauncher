package bootstrap_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"plumelauncher/internal/bootstrap"
)

func TestInitializeDefaultRoot(t *testing.T) {
	// Clear env to use default
	t.Setenv("PLUME_DATA_ROOT", "")

	cfg, err := bootstrap.Initialize("")
	if err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	if cfg.DataRoot == "" {
		t.Fatal("DataRoot is empty")
	}
	if cfg.LogDir == "" {
		t.Fatal("LogDir is empty")
	}

	// Verify directory was created
	if _, err := os.Stat(cfg.DataRoot); os.IsNotExist(err) {
		t.Errorf("DataRoot dir was not created: %s", cfg.DataRoot)
	}

	// Verify log dir was created
	logDir := filepath.Join(cfg.DataRoot, "logs")
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		t.Errorf("LogDir was not created: %s", logDir)
	}

	// Verify platform-specific default
	if runtime.GOOS == "windows" {
		if filepath.Base(cfg.DataRoot) != "PlumeLauncher" {
			t.Errorf("DataRoot base = %q, want %q", filepath.Base(cfg.DataRoot), "PlumeLauncher")
		}
	}
}

func TestInitializeCustomRoot(t *testing.T) {
	custom := filepath.Join(t.TempDir(), "custom-root")

	cfg, err := bootstrap.Initialize(custom)
	if err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	if cfg.DataRoot != custom {
		t.Errorf("DataRoot = %q, want %q", cfg.DataRoot, custom)
	}
}

func TestInitializeEnvOverride(t *testing.T) {
	envRoot := filepath.Join(t.TempDir(), "env-root")
	t.Setenv("PLUME_DATA_ROOT", envRoot)

	cfg, err := bootstrap.Initialize("")
	if err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	if cfg.DataRoot != envRoot {
		t.Errorf("DataRoot = %q, want %q", cfg.DataRoot, envRoot)
	}
}
