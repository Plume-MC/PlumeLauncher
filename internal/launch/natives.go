package launch

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ExtractNatives extracts native JAR archives into targetDir.
// Rejects paths that attempt directory traversal.
func ExtractNatives(jarPath string, targetDir string) error {
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return err
	}

	r, err := zip.OpenReader(jarPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}

		name := f.Name

		// Skip META-INF
		if strings.HasPrefix(name, "META-INF/") {
			continue
		}

		// Skip non-binary files
		if strings.HasSuffix(name, ".sha1") ||
			strings.HasSuffix(name, ".md5") ||
			strings.HasSuffix(name, ".txt") {
			continue
		}

		destPath := filepath.Join(targetDir, name)

		// Path traversal rejection
		cleanDest := filepath.Clean(destPath)
		cleanTarget := filepath.Clean(targetDir)
		if !strings.HasPrefix(cleanDest, cleanTarget+string(os.PathSeparator)) &&
			cleanDest != cleanTarget {
			return fmt.Errorf("path traversal detected: %s", name)
		}

		if err := extractFile(f, destPath); err != nil {
			return err
		}
	}

	return nil
}

func extractFile(f *zip.File, destPath string) error {
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return err
	}

	outFile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	_, err = io.Copy(outFile, rc)
	return err
}
