package services

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"plumelauncher/internal/instances"
	"plumelauncher/internal/metadata"
)

func writeFakeManagedJDK(t *testing.T, dataRoot string, major int, version string) {
	t.Helper()
	binDir := filepath.Join(dataRoot, "runtimes", "java", "21", "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\necho 'openjdk version \"" + version + "\" 2024-01-16'\n"
	if err := os.WriteFile(filepath.Join(binDir, "java"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{"major":` + strconv.Itoa(major) + `,"version":"` + version + `"}`
	if err := os.WriteFile(filepath.Join(dataRoot, "runtimes", "java", "21", "manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLaunchJavaCandidatesIncludesManaged(t *testing.T) {
	dir := t.TempDir()
	writeFakeManagedJDK(t, dir, 21, "21.0.2")

	candidates, err := launchJavaCandidates(dir, instances.DefaultLauncherDefaults())
	if err != nil {
		t.Fatalf("launchJavaCandidates: %v", err)
	}
	for _, c := range candidates {
		if c.Major == 21 && strings.HasPrefix(c.Path, dir) {
			return
		}
	}
	t.Fatalf("managed JDK missing from candidates: %+v", candidates)
}

func TestLaunchRequiredMajorPrefersDetailJavaVersion(t *testing.T) {
	loader := metadata.VersionDetail{ID: "fabric-loader-0.16.14-1.20.1"}
	if got := launchRequiredMajor(loader, "1.20.1"); got != 17 {
		t.Fatalf("loader detail required = %d, want 17", got)
	}
	withField := metadata.VersionDetail{
		ID:          "26.2",
		JavaVersion: metadata.JavaVersion{Component: "jre-legacy", MajorVersion: 25},
	}
	if got := launchRequiredMajor(withField, "26.2"); got != 25 {
		t.Fatalf("detail javaVersion required = %d, want 25", got)
	}
	if got := launchRequiredMajor(metadata.VersionDetail{ID: "1.16.5"}, "1.16.5"); got != 8 {
		t.Fatalf("plain detail required = %d, want 8", got)
	}
}
