package bootstrap

import (
	"os"
	"path/filepath"
	"testing"
)

func writeMarker(t *testing.T, dir string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, portableMarkerName()), []byte{}, 0o644); err != nil {
		t.Fatalf("write marker: %v", err)
	}
}

func TestPortableMarkerWinsOverDefault(t *testing.T) {
	exeDir := t.TempDir()
	writeMarker(t, exeDir)

	root, err := resolveGameRoot("", "", exeDir, func() (string, error) { return "", nil })
	if err != nil {
		t.Fatalf("resolveGameRoot: %v", err)
	}
	if root != exeDir {
		t.Errorf("root = %q, want exe dir %q", root, exeDir)
	}
}

func TestPortableMarkerWinsOverBootstrapJSON(t *testing.T) {
	exeDir := t.TempDir()
	writeMarker(t, exeDir)

	root, err := resolveGameRoot("", "", exeDir, func() (string, error) { return filepath.Join("some", "saved-root"), nil })
	if err != nil {
		t.Fatalf("resolveGameRoot: %v", err)
	}
	if root != exeDir {
		t.Errorf("root = %q, want exe dir %q", root, exeDir)
	}
}

func TestPortableMarkerLosesToEnvAndCustomRoot(t *testing.T) {
	exeDir := t.TempDir()
	writeMarker(t, exeDir)
	configured := func() (string, error) { return filepath.Join("some", "saved-root"), nil }

	if root, err := resolveGameRoot(filepath.Join("custom", "root"), filepath.Join("env", "root"), exeDir, configured); err != nil || root != filepath.Join("custom", "root") {
		t.Errorf("customRoot: root = %q, err = %v", root, err)
	}
	if root, err := resolveGameRoot("", filepath.Join("env", "root"), exeDir, configured); err != nil || root != filepath.Join("env", "root") {
		t.Errorf("env: root = %q, err = %v", root, err)
	}
}

func TestIsPortableRootOnlyWithMarkerBesideExe(t *testing.T) {
	exeDir := t.TempDir()
	if isPortableRoot(exeDir, exeDir) {
		t.Error("isPortableRoot = true without marker")
	}
	writeMarker(t, exeDir)
	if !isPortableRoot(exeDir, exeDir) {
		t.Error("isPortableRoot = false with marker beside exe dir")
	}
	if isPortableRoot(filepath.Join(t.TempDir(), "elsewhere"), exeDir) {
		t.Error("isPortableRoot = true for a different data root")
	}
	if isPortableRoot(exeDir, "") {
		t.Error("isPortableRoot = true with unknown exe dir")
	}
}

func TestNoMarkerKeepsExistingBehavior(t *testing.T) {
	exeDir := t.TempDir()

	root, err := resolveGameRoot("", "", exeDir, func() (string, error) { return filepath.Join("some", "saved-root"), nil })
	if err != nil {
		t.Fatalf("resolveGameRoot: %v", err)
	}
	if root != filepath.Join("some", "saved-root") {
		t.Errorf("root = %q, want saved bootstrap root", root)
	}
}
