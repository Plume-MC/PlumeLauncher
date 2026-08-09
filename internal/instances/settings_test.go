package instances_test

import (
	"testing"

	"plumelauncher/internal/instances"
)

func TestEffectiveSettingsFromDefaults(t *testing.T) {
	defaults := instances.DefaultLauncherDefaults()
	inst := instances.Instance{
		ID: "test",
	}

	result := instances.EffectiveSettings(inst, defaults)

	if result.MinRamMB != 1024 {
		t.Errorf("MinRamMB = %d, want 1024", result.MinRamMB)
	}
	if result.MaxRamMB != 4096 {
		t.Errorf("MaxRamMB = %d, want 4096", result.MaxRamMB)
	}
	if result.ResolutionW != 854 {
		t.Errorf("ResolutionW = %d, want 854", result.ResolutionW)
	}
	if result.GPUPreference != "auto" {
		t.Errorf("GPUPreference = %q, want %q", result.GPUPreference, "auto")
	}
}

func TestEffectiveSettingsOverrides(t *testing.T) {
	defaults := instances.DefaultLauncherDefaults()
	inst := instances.Instance{
		ID: "test",
		Settings: instances.Settings{
			MinRamMB:     instances.IntPtr(2048),
			MaxRamMB:     instances.IntPtr(8192),
			ResolutionW:  instances.IntPtr(1920),
			ResolutionH:  instances.IntPtr(1080),
			JavaPath:     instances.StringPtr("/custom/java"),
			GPUPreference: instances.StringPtr("discrete"),
		},
	}

	result := instances.EffectiveSettings(inst, defaults)

	if result.MinRamMB != 2048 {
		t.Errorf("MinRamMB = %d, want 2048", result.MinRamMB)
	}
	if result.MaxRamMB != 8192 {
		t.Errorf("MaxRamMB = %d, want 8192", result.MaxRamMB)
	}
	if result.ResolutionW != 1920 {
		t.Errorf("ResolutionW = %d, want 1920", result.ResolutionW)
	}
	if result.JavaPath != "/custom/java" {
		t.Errorf("JavaPath = %q, want %q", result.JavaPath, "/custom/java")
	}
	if result.GPUPreference != "discrete" {
		t.Errorf("GPUPreference = %q, want %q", result.GPUPreference, "discrete")
	}
}

func TestEffectiveSettingsPartialOverride(t *testing.T) {
	defaults := instances.DefaultLauncherDefaults()
	inst := instances.Instance{
		ID: "test",
		Settings: instances.Settings{
			MinRamMB: instances.IntPtr(2048),
			// MaxRamMB not set — should inherit from defaults
		},
	}

	result := instances.EffectiveSettings(inst, defaults)

	if result.MinRamMB != 2048 {
		t.Errorf("MinRamMB = %d, want 2048", result.MinRamMB)
	}
	if result.MaxRamMB != 4096 {
		t.Errorf("MaxRamMB = %d, want 4096 (inherit)", result.MaxRamMB)
	}
}

func TestEffectiveSettingsEmptyStringInherits(t *testing.T) {
	defaults := instances.DefaultLauncherDefaults()
	inst := instances.Instance{
		ID: "test",
		Settings: instances.Settings{
			JavaPath: instances.StringPtr(""),
		},
	}

	result := instances.EffectiveSettings(inst, defaults)

	if result.JavaPath != "" {
		t.Errorf("JavaPath = %q, want empty (inherit)", result.JavaPath)
	}
}

func TestDefaultLauncherDefaults(t *testing.T) {
	defaults := instances.DefaultLauncherDefaults()

	if defaults.Theme != "dark" {
		t.Errorf("Theme = %q, want %q", defaults.Theme, "dark")
	}
	if defaults.DefaultMinRamMB != 1024 {
		t.Errorf("DefaultMinRamMB = %d, want 1024", defaults.DefaultMinRamMB)
	}
	if defaults.DefaultMaxRamMB != 4096 {
		t.Errorf("DefaultMaxRamMB = %d, want 4096", defaults.DefaultMaxRamMB)
	}
}
