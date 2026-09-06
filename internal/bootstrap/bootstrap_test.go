package bootstrap_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"plumelauncher/internal/bootstrap"
)

func TestInitializeDefaultRoot(t *testing.T) {
	t.Setenv("PLUME_DATA_ROOT", "")

	cfg, err := bootstrap.Initialize("")
	if err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	if cfg.DataRoot == "" {
		t.Fatal("DataRoot is empty")
	}
	if _, err := os.Stat(cfg.DataRoot); os.IsNotExist(err) {
		t.Errorf("DataRoot dir was not created: %s", cfg.DataRoot)
	}
	if runtime.GOOS == "windows" {
		if filepath.Base(cfg.DataRoot) != "PlumeLauncher" {
			t.Errorf("DataRoot base = %q, want %q", filepath.Base(cfg.DataRoot), "PlumeLauncher")
		}
	}
}

func TestInitializeCustomRoot(t *testing.T) {
	base := t.TempDir()
	t.Setenv("LOCALAPPDATA", base)
	t.Setenv("USERPROFILE", base)
	t.Setenv("XDG_DATA_HOME", base)
	t.Setenv("PLUME_DATA_ROOT", "")

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
	base := t.TempDir()
	t.Setenv("LOCALAPPDATA", base)
	t.Setenv("USERPROFILE", base)
	t.Setenv("XDG_DATA_HOME", base)
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

func TestSetDataRootAppliesOnNextInitialize(t *testing.T) {
	base := t.TempDir()
	t.Setenv("LOCALAPPDATA", base)
	t.Setenv("USERPROFILE", base)
	t.Setenv("XDG_DATA_HOME", base)
	t.Setenv("PLUME_DATA_ROOT", "")
	target := filepath.Join(t.TempDir(), "next-root")

	if err := bootstrap.SetDataRoot(target); err != nil {
		t.Fatalf("SetDataRoot: %v", err)
	}
	config, err := bootstrap.Initialize("")
	if err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	if config.DataRoot != target {
		t.Errorf("DataRoot = %q, want %q", config.DataRoot, target)
	}
	if _, err := os.Stat(filepath.Join(base, "PlumeLauncher", "bootstrap.json")); err != nil {
		t.Fatalf("bootstrap.json missing under default bootstrap root: %v", err)
	}
}
