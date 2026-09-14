package cleanup

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCleanupOrphanedPartFiles_RemovesOldFiles(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "cache", "java")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create old .part file (simulate interrupted download)
	oldPart := filepath.Join(cacheDir, "OpenJDK21U.tar.gz.part")
	if err := os.WriteFile(oldPart, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Backdate the modification time to 2 hours ago
	os.Chtimes(oldPart, time.Now().Add(-2*time.Hour), time.Now().Add(-2*time.Hour))

	// Create recent .part file (should be kept)
	recentPart := filepath.Join(cacheDir, "OpenJDK17U.tar.gz.part")
	if err := os.WriteFile(recentPart, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create non-.part file (should be kept)
	other := filepath.Join(cacheDir, "OpenJDK25U.tar.gz")
	if err := os.WriteFile(other, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	removed := CleanupOrphanedPartFiles(dir)
	if removed != 1 {
		t.Errorf("expected 1 removed, got %d", removed)
	}
	if _, err := os.Stat(oldPart); !os.IsNotExist(err) {
		t.Error("old .part file should have been removed")
	}
	if _, err := os.Stat(recentPart); err != nil {
		t.Error("recent .part file should have been kept")
	}
	if _, err := os.Stat(other); err != nil {
		t.Error("non-.part file should have been kept")
	}
}

func TestCleanupOrphanedPartFiles_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	removed := CleanupOrphanedPartFiles(dir)
	if removed != 0 {
		t.Errorf("expected 0 removed for empty dir, got %d", removed)
	}
}

func TestCleanupOrphanedPartFiles_NoCacheDir(t *testing.T) {
	dir := t.TempDir()
	removed := CleanupOrphanedPartFiles(dir)
	if removed != 0 {
		t.Errorf("expected 0 removed for missing cache dir, got %d", removed)
	}
}

func TestCleanupPartFiles_ThresholdBoundary(t *testing.T) {
	dir := t.TempDir()

	// File at exactly the boundary (1 hour old)
	part := filepath.Join(dir, "test.tar.gz.part")
	if err := os.WriteFile(part, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	os.Chtimes(part, time.Now().Add(-maxPartFileAge), time.Now().Add(-maxPartFileAge))

	removed := cleanupPartFiles(dir, maxPartFileAge)
	if removed != 0 {
		t.Errorf("file at exact boundary should not be removed, got %d", removed)
	}
}
