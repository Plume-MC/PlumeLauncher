package downloader_test

import (
	"os"
	"path/filepath"
	"testing"

	"plumelauncher/internal/downloader"
)

func TestVerifyFileSHA1Valid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("hello world"), 0o644)

	// SHA1 of "hello world" = 2aae6c35c94fcfb415dbe95f408b9ce91ee846ed
	ok, err := downloader.VerifyFileSHA1(path, "2aae6c35c94fcfb415dbe95f408b9ce91ee846ed")
	if err != nil {
		t.Fatalf("VerifyFileSHA1: %v", err)
	}
	if !ok {
		t.Error("expected valid hash")
	}
}

func TestVerifyFileSHA1Invalid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("hello world"), 0o644)

	ok, err := downloader.VerifyFileSHA1(path, "0000000000000000000000000000000000000000")
	if err != nil {
		t.Fatalf("VerifyFileSHA1: %v", err)
	}
	if ok {
		t.Error("expected invalid hash")
	}
}

func TestVerifyFileSHA1EmptyExpected(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("data"), 0o644)

	ok, err := downloader.VerifyFileSHA1(path, "")
	if err != nil {
		t.Fatalf("VerifyFileSHA1: %v", err)
	}
	if !ok {
		t.Error("empty expected should skip verification")
	}
}

func TestVerifyFileSHA1MissingFile(t *testing.T) {
	_, err := downloader.VerifyFileSHA1("/nonexistent/file.txt", "abc")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestCommitFileSuccess(t *testing.T) {
	dir := t.TempDir()
	partPath := filepath.Join(dir, "file.txt.part")
	finalPath := filepath.Join(dir, "file.txt")
	os.WriteFile(partPath, []byte("content"), 0o644)

	err := downloader.CommitFile(partPath, finalPath, "", 7)
	if err != nil {
		t.Fatalf("CommitFile: %v", err)
	}

	data, _ := os.ReadFile(finalPath)
	if string(data) != "content" {
		t.Errorf("content = %q, want %q", string(data), "content")
	}
	if _, err := os.Stat(partPath); !os.IsNotExist(err) {
		t.Error(".part file should be removed after commit")
	}
}

func TestCommitFileWithHash(t *testing.T) {
	dir := t.TempDir()
	partPath := filepath.Join(dir, "file.txt.part")
	finalPath := filepath.Join(dir, "file.txt")
	os.WriteFile(partPath, []byte("hello world"), 0o644)

	err := downloader.CommitFile(partPath, finalPath, "2aae6c35c94fcfb415dbe95f408b9ce91ee846ed", 11)
	if err != nil {
		t.Fatalf("CommitFile: %v", err)
	}

	if _, err := os.Stat(finalPath); os.IsNotExist(err) {
		t.Error("final file should exist")
	}
}

func TestCommitFileHashMismatch(t *testing.T) {
	dir := t.TempDir()
	partPath := filepath.Join(dir, "file.txt.part")
	finalPath := filepath.Join(dir, "file.txt")
	os.WriteFile(partPath, []byte("wrong data"), 0o644)

	err := downloader.CommitFile(partPath, finalPath, "2aae6c35c94fcfb415dbe95f408b9ce91ee846ed", 10)
	if err != downloader.ErrHashMismatch {
		t.Fatalf("expected ErrHashMismatch, got %v", err)
	}
	if _, err := os.Stat(partPath); !os.IsNotExist(err) {
		t.Error(".part file should be removed on hash mismatch")
	}
	if _, err := os.Stat(finalPath); !os.IsNotExist(err) {
		t.Error("final file should not exist on hash mismatch")
	}
}

func TestCommitFileCreatesNestedDir(t *testing.T) {
	dir := t.TempDir()
	partPath := filepath.Join(dir, "file.txt.part")
	finalPath := filepath.Join(dir, "sub", "dir", "file.txt")
	os.WriteFile(partPath, []byte("data"), 0o644)

	err := downloader.CommitFile(partPath, finalPath, "", 4)
	if err != nil {
		t.Fatalf("CommitFile: %v", err)
	}

	data, _ := os.ReadFile(finalPath)
	if string(data) != "data" {
		t.Errorf("content = %q", string(data))
	}
}

func TestCommitFileRejectsMissingIntegrityMetadata(t *testing.T) {
	dir := t.TempDir()
	partPath := filepath.Join(dir, "file.txt.part")
	finalPath := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(partPath, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := downloader.CommitFile(partPath, finalPath, "", 0)
	if err != downloader.ErrMissingIntegrityMetadata {
		t.Fatalf("expected ErrMissingIntegrityMetadata, got %v", err)
	}
	if _, err := os.Stat(finalPath); !os.IsNotExist(err) {
		t.Fatal("final file should not exist without integrity metadata")
	}
}
