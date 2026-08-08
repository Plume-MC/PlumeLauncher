package launch_test

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"plumelauncher/internal/launch"
)

func createTestJar(t *testing.T, dir string, files map[string]string) string {
	t.Helper()
	jarPath := filepath.Join(dir, "test.jar")

	f, err := os.Create(jarPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	w := zip.NewWriter(f)
	for name, content := range files {
		fw, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		fw.Write([]byte(content))
	}
	w.Close()

	return jarPath
}

func TestExtractNativesBasic(t *testing.T) {
	dir := t.TempDir()
	jarPath := createTestJar(t, dir, map[string]string{
		"lib.dll":        "binary",
		"lib.so":         "binary",
		"META-INF/MANIFEST.MF": "Manifest-Version: 1.0",
		"readme.txt":     "text",
	})

	targetDir := filepath.Join(dir, "natives")
	err := launch.ExtractNatives(jarPath, targetDir)
	if err != nil {
		t.Fatalf("ExtractNatives: %v", err)
	}

	// lib.dll should be extracted
	if _, err := os.Stat(filepath.Join(targetDir, "lib.dll")); os.IsNotExist(err) {
		t.Error("lib.dll not extracted")
	}

	// META-INF should be skipped
	if _, err := os.Stat(filepath.Join(targetDir, "META-INF")); !os.IsNotExist(err) {
		t.Error("META-INF should be skipped")
	}

	// readme.txt should be skipped
	if _, err := os.Stat(filepath.Join(targetDir, "readme.txt")); !os.IsNotExist(err) {
		t.Error("readme.txt should be skipped")
	}
}

func TestExtractNativesPathTraversal(t *testing.T) {
	dir := t.TempDir()
	jarPath := createTestJar(t, dir, map[string]string{
		"../../etc/passwd": "malicious",
	})

	targetDir := filepath.Join(dir, "natives")
	err := launch.ExtractNatives(jarPath, targetDir)
	if err == nil {
		t.Fatal("expected path traversal error")
	}
}

func TestExtractNativesSubdirs(t *testing.T) {
	dir := t.TempDir()
	jarPath := createTestJar(t, dir, map[string]string{
		"native.dll":         "binary",
		"sub/dir/native.dll": "binary",
	})

	targetDir := filepath.Join(dir, "natives")
	err := launch.ExtractNatives(jarPath, targetDir)
	if err != nil {
		t.Fatalf("ExtractNatives: %v", err)
	}

	if _, err := os.Stat(filepath.Join(targetDir, "native.dll")); os.IsNotExist(err) {
		t.Error("native.dll not extracted")
	}
	if _, err := os.Stat(filepath.Join(targetDir, "sub", "dir", "native.dll")); os.IsNotExist(err) {
		t.Error("sub/dir/native.dll not extracted")
	}
}
