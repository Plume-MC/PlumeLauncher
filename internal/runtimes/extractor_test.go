package runtimes

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestStripTopLevelDir(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"jdk-25.0.4+1/bin/java", "bin/java"},
		{"jdk-25.0.4+1/lib/modules", "lib/modules"},
		{"top-level-only", ""},
		{"", ""},
		{"a/b", "b"},
	}

	for _, tt := range tests {
		got := stripTopLevelDir(tt.input)
		if got != tt.want {
			t.Errorf("stripTopLevelDir(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestExtractZip(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "test.zip")
	destDir := filepath.Join(dir, "extracted")

	// Create a test zip with a top-level directory
	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)

	// Add a file inside a top-level dir
	entry, err := w.Create("jdk-test/bin/java")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte("#!/bin/sh\necho java")); err != nil {
		t.Fatal(err)
	}

	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	f.Close()

	// Extract
	if err := ExtractJDK(archivePath, destDir); err != nil {
		t.Fatalf("ExtractJDK failed: %v", err)
	}

	// Verify the file was extracted with top-level dir stripped
	extracted := filepath.Join(destDir, "bin", "java")
	data, err := os.ReadFile(extracted)
	if err != nil {
		t.Fatalf("extracted file not found: %v", err)
	}
	if string(data) != "#!/bin/sh\necho java" {
		t.Errorf("unexpected content: %s", string(data))
	}
}

func TestExtractTarGz(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "test.tar.gz")
	destDir := filepath.Join(dir, "extracted")

	// Create a test tar.gz
	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)

	// Add a file inside a top-level dir
	content := []byte("#!/bin/sh\necho java")
	header := &tar.Header{
		Name: "jdk-test/bin/java",
		Mode: 0o755,
		Size: int64(len(content)),
	}
	if err := tw.WriteHeader(header); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatal(err)
	}

	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	f.Close()

	// Extract
	if err := ExtractJDK(archivePath, destDir); err != nil {
		t.Fatalf("ExtractJDK failed: %v", err)
	}

	// Verify
	extracted := filepath.Join(destDir, "bin", "java")
	data, err := os.ReadFile(extracted)
	if err != nil {
		t.Fatalf("extracted file not found: %v", err)
	}
	if string(data) != "#!/bin/sh\necho java" {
		t.Errorf("unexpected content: %s", string(data))
	}
}

func TestExtractTarGz_SkipsLegalSymlinks(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "test.tar.gz")
	destDir := filepath.Join(dir, "extracted")

	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)

	if err := tw.WriteHeader(&tar.Header{
		Name:     "jdk-test/legal/jdk.crypto.ec/ASSEMBLY_EXCEPTION",
		Linkname: "../java.base/ASSEMBLY_EXCEPTION",
		Typeflag: tar.TypeSymlink,
	}); err != nil {
		t.Fatal(err)
	}

	content := []byte("#!/bin/sh\necho java")
	if err := tw.WriteHeader(&tar.Header{
		Name: "jdk-test/bin/java",
		Mode: 0o755,
		Size: int64(len(content)),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatal(err)
	}

	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	if err := ExtractJDK(archivePath, destDir); err != nil {
		t.Fatalf("ExtractJDK failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(destDir, "bin", "java")); err != nil {
		t.Fatalf("runtime file was not extracted: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(destDir, "legal", "jdk.crypto.ec", "ASSEMBLY_EXCEPTION")); !os.IsNotExist(err) {
		t.Fatalf("legal symlink should be skipped, got error: %v", err)
	}
}

func TestExtractJDK_UnsupportedFormat(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "test.bin")
	destDir := filepath.Join(dir, "extracted")

	os.WriteFile(archivePath, []byte("not an archive"), 0o644)

	err := ExtractJDK(archivePath, destDir)
	if err == nil {
		t.Error("expected error for unsupported format")
	}
}

func TestExtractZip_TraversalRejected(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "evil.zip")
	destDir := filepath.Join(dir, "extracted")

	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)

	// Try path traversal
	entry, err := w.Create("jdk-evil/../../etc/passwd")
	if err != nil {
		t.Fatal(err)
	}
	entry.Write([]byte("evil"))

	w.Close()
	f.Close()

	err = ExtractJDK(archivePath, destDir)
	if err == nil {
		t.Error("expected path traversal error")
	}
}

func TestExtractTarGz_TraversalRejected(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "evil.tar.gz")
	destDir := filepath.Join(dir, "extracted")

	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)

	content := []byte("evil")
	header := &tar.Header{
		Name: "jdk-evil/../../etc/passwd",
		Mode: 0o644,
		Size: int64(len(content)),
	}
	tw.WriteHeader(header)
	tw.Write(content)
	tw.Close()
	gz.Close()
	f.Close()

	err = ExtractJDK(archivePath, destDir)
	if err == nil {
		t.Error("expected path traversal error")
	}
}
