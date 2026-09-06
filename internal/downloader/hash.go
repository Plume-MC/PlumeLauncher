package downloader

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var (
	// ErrHashMismatch is returned when a downloaded file's SHA1 does not match expected.
	ErrHashMismatch = errors.New("sha1 hash mismatch")
	ErrSizeMismatch = errors.New("file size mismatch")
	// ErrMissingIntegrityMetadata prevents unverified artifacts from being committed.
	ErrMissingIntegrityMetadata = errors.New("artifact integrity metadata missing")
	ErrInvalidIntegrityMetadata = errors.New("invalid artifact integrity metadata")
	ErrHashVerificationRequired = errors.New("hash verification required")
)

func validateIntegrityMetadata(expectedSHA1 string, expectedSize int64) error {
	if expectedSize < 0 {
		return ErrInvalidIntegrityMetadata
	}
	if expectedSHA1 == "" && expectedSize == 0 {
		return ErrMissingIntegrityMetadata
	}
	if expectedSHA1 != "" {
		if len(expectedSHA1) != sha1.Size*2 {
			return fmt.Errorf("%w: SHA-1 must be %d hexadecimal characters", ErrInvalidIntegrityMetadata, sha1.Size*2)
		}
		if _, err := hex.DecodeString(expectedSHA1); err != nil {
			return fmt.Errorf("%w: SHA-1 is not hexadecimal", ErrInvalidIntegrityMetadata)
		}
	}
	return nil
}

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

// CommitFile verifies a .part file's expected size and SHA1, then atomically moves it to finalPath.
// At least one integrity signal (size or SHA1) is required.
func CommitFile(partPath string, finalPath string, expectedSHA1 string, expectedSize int64) error {
	if err := validateIntegrityMetadata(expectedSHA1, expectedSize); err != nil {
		return err
	}
	if expectedSize > 0 {
		info, err := os.Stat(partPath)
		if err != nil {
			return err
		}
		if info.Size() != expectedSize {
			os.Remove(partPath)
			return ErrSizeMismatch
		}
	}

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

	// Rename only after validation. If a platform cannot replace an existing file,
	// leave both the existing valid artifact and the verified .part untouched.
	if err := os.Rename(partPath, finalPath); err != nil {
		return err
	}

	return nil
}
