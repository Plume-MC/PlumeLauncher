package services

import (
	"testing"

	"plumelauncher/internal/instances"
	"plumelauncher/internal/metadata"
)

func TestJavaVersionForLaunchUsesParentGameVersion(t *testing.T) {
	detail := metadata.VersionDetail{ID: "fabric-loader-0.16.0-1.20.1", Jar: "1.20.1"}
	if got := javaVersionForLaunch(detail); got != "1.20.1" {
		t.Fatalf("javaVersionForLaunch() = %q, want 1.20.1", got)
	}
}

func TestJavaVersionForLaunchUsesVersionIDForVanilla(t *testing.T) {
	detail := metadata.VersionDetail{ID: "1.21.4"}
	if got := javaVersionForLaunch(detail); got != "1.21.4" {
		t.Fatalf("javaVersionForLaunch() = %q, want 1.21.4", got)
	}
}

func TestLaunchServiceStopCancelsBeforeProcessIsRegistered(t *testing.T) {
	registry := instances.NewRegistry()
	if _, err := registry.Start("instance", instances.OpLaunch); err != nil {
		t.Fatal(err)
	}

	service := &LaunchService{Registry: registry}
	if err := service.Stop("instance"); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if registry.IsActive("instance") {
		t.Fatal("launch operation remains active")
	}
}

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
