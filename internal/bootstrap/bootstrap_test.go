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

	if cfg.AppRoot == "" {
		t.Fatal("AppRoot is empty")
	}
	if cfg.GameRoot == "" {
		t.Fatal("GameRoot is empty")
	}
	if _, err := os.Stat(cfg.AppRoot); os.IsNotExist(err) {
		t.Errorf("AppRoot dir was not created: %s", cfg.AppRoot)
	}
	if _, err := os.Stat(cfg.GameRoot); os.IsNotExist(err) {
		t.Errorf("GameRoot dir was not created: %s", cfg.GameRoot)
	}
	if runtime.GOOS == "windows" {
		if filepath.Base(cfg.AppRoot) != "PlumeLauncher" {
			t.Errorf("AppRoot base = %q, want %q", filepath.Base(cfg.AppRoot), "PlumeLauncher")
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

	if cfg.GameRoot != custom {
		t.Errorf("GameRoot = %q, want %q", cfg.GameRoot, custom)
	}
	if cfg.AppRoot != filepath.Join(base, "PlumeLauncher") && cfg.AppRoot != filepath.Join(base, "PlumeLauncher") {
		// AppRoot always fixed under env base
		if filepath.Base(cfg.AppRoot) != "PlumeLauncher" {
			t.Errorf("AppRoot = %q, want fixed app root", cfg.AppRoot)
		}
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

	if cfg.GameRoot != envRoot {
		t.Errorf("GameRoot = %q, want %q", cfg.GameRoot, envRoot)
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
	if config.GameRoot != target {
		t.Errorf("GameRoot = %q, want %q", config.GameRoot, target)
	}
	if config.AppRoot != filepath.Join(base, "PlumeLauncher") {
		// linux uses XDG under base
		if filepath.Base(config.AppRoot) != "PlumeLauncher" {
			t.Errorf("AppRoot drifted: %q", config.AppRoot)
		}
	}
	if _, err := os.Stat(filepath.Join(config.AppRoot, "bootstrap.json")); err != nil {
		t.Fatalf("bootstrap.json missing under AppRoot: %v", err)
	}
}
