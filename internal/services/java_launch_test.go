package services

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"plumelauncher/internal/instances"
	"plumelauncher/internal/java"
	"plumelauncher/internal/metadata"
)

func writeFakeManagedJDK(t *testing.T, dataRoot string, major int, version string) {
	t.Helper()
	binDir := filepath.Join(dataRoot, "runtimes", "java", "21", "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Only the file presence matters: ListManaged looks for java.exe on
	// Windows and java elsewhere, while validation is stubbed below so the
	// file never needs to be executable on any OS.
	name := "java"
	if runtime.GOOS == "windows" {
		name = "java.exe"
	}
	if err := os.WriteFile(filepath.Join(binDir, name), []byte("fake-jdk-stub"), 0o755); err != nil {
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

	origScan := scanJavaInstallations
	origValidate := validateJavaPath
	t.Cleanup(func() {
		scanJavaInstallations = origScan
		validateJavaPath = origValidate
	})
	scanJavaInstallations = func() ([]java.JavaInfo, error) {
		return nil, nil
	}
	validateJavaPath = func(path string, requiredMajor int) (*java.JavaInfo, error) {
		if strings.HasPrefix(path, dir) {
			return &java.JavaInfo{Path: path, Version: "21.0.2", Major: 21}, nil
		}
		return nil, os.ErrNotExist
	}

	candidates, err := launchJavaCandidates(dir, instances.DefaultLauncherDefaults())
	if err != nil {
		t.Fatalf("launchJavaCandidates: %v", err)
	}
	wantDir := dir
	if resolved, err := filepath.EvalSymlinks(dir); err == nil {
		wantDir = resolved
	}
	for _, c := range candidates {
		if c.Major == 21 && (strings.HasPrefix(c.Path, dir) || strings.HasPrefix(c.Path, wantDir)) {
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
