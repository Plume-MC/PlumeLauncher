package services

import "testing"

func TestLaunchLogLevel(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		isStderr bool
		want     string
	}{
		{"stdout info", "Starting Minecraft", false, "info"},
		{"java warning", "WARNING: A restricted method has been called", true, "warning"},
		{"unmarked stderr", "native output", true, "warning"},
		{"explicit error", "[ERROR] Missing library", true, "error"},
		{"exception", "java.lang.RuntimeException: failed", true, "error"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := launchLogLevel(test.line, test.isStderr); got != test.want {
				t.Fatalf("launchLogLevel(%q, %t) = %q, want %q", test.line, test.isStderr, got, test.want)
			}
		})
	}
}
