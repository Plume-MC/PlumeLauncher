package launch_test

import (
	"testing"

	"plumelauncher/internal/launch"
)

func TestWindowsGPUPreferenceValue(t *testing.T) {
	tests := map[string]string{
		"auto":       "GpuPreference=0;",
		"integrated": "GpuPreference=1;",
		"discrete":   "GpuPreference=2;",
	}
	for preference, want := range tests {
		if got := launch.WindowsGPUPreferenceValue(preference); got != want {
			t.Errorf("WindowsGPUPreferenceValue(%q) = %q, want %q", preference, got, want)
		}
	}
}
