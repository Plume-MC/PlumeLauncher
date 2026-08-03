package downloader

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ErrHashMismatch is returned when a downloaded file's SHA1 does not match expected.
var ErrHashMismatch = errors.New("sha1 hash mismatch")

// VerifyFileSHA1 checks if a file matches the expected SHA1 hash.
// Returns (true, nil) if expected is empty (skip verification).
func VerifyFileSHA1(path string, expected string) (bool, error) {
	if expected == "" {
		return true, nil
	}

	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	h := sha1.New()
	if _, err := io.Copy(h, f); err != nil {
		return false, err
	}

	actual := hex.EncodeToString(h.Sum(nil))
	return strings.EqualFold(actual, expected), nil
}

// CommitFile verifies a .part file's SHA1, then atomically moves it to finalPath.
// If expectedSHA1 is empty, hash verification is skipped.
// On any error, the .part file is cleaned up.
func CommitFile(partPath string, finalPath string, expectedSHA1 string) error {
	// Verify hash if expected is provided
	if expectedSHA1 != "" {
		ok, err := VerifyFileSHA1(partPath, expectedSHA1)
		if err != nil {
			os.Remove(partPath)
			return err
		}
		if !ok {
			os.Remove(partPath)
			return ErrHashMismatch
		}
	}

	// Ensure target directory exists
	dir := filepath.Dir(finalPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		os.Remove(partPath)
		return err
	}

	// Remove existing final file if present
	os.Remove(finalPath)

	// Atomic rename
	if err := os.Rename(partPath, finalPath); err != nil {
		os.Remove(partPath)
		return err
	}

	return nil
}
