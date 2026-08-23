package launch

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"plumelauncher/internal/security"
)

const (
	maxNativeEntries    = 4096
	maxNativeFileBytes  = 64 << 20
	maxNativeTotalBytes = 256 << 20
)

// ExtractNatives extracts native JAR archives into targetDir.
// Rejects paths that attempt directory traversal.
func ExtractNatives(jarPath string, targetDir string, excludes ...[]string) error {
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return err
	}

	r, err := zip.OpenReader(jarPath)
	if err != nil {
		return err
	}
	defer r.Close()
	if len(r.File) > maxNativeEntries {
		return fmt.Errorf("native archive has too many entries: %d", len(r.File))
	}
	var totalBytes uint64
	var excluded []string
	if len(excludes) > 0 {
		excluded = excludes[0]
	}

	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		if f.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink entry not allowed: %s", f.Name)
		}

		name := filepath.ToSlash(f.Name)

		if isExcludedNative(name, excluded) {
			continue
		}
		if f.UncompressedSize64 > maxNativeFileBytes || totalBytes > maxNativeTotalBytes-f.UncompressedSize64 {
			return fmt.Errorf("native archive exceeds extraction limit: %s", name)
		}
		totalBytes += f.UncompressedSize64

		// Skip non-binary files
		if strings.HasSuffix(name, ".sha1") ||
			strings.HasSuffix(name, ".md5") ||
			strings.HasSuffix(name, ".txt") {
			continue
		}

		destPath, err := security.ResolveUnderRoot(targetDir, name)
		if err != nil {
			return fmt.Errorf("path traversal detected: %s", name)
		}
		if info, statErr := os.Lstat(destPath); statErr == nil && info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("destination symlink not allowed: %s", name)
		}

		if err := extractFile(f, destPath); err != nil {
			return err
		}
	}

	return nil
}

func isExcludedNative(name string, excludes []string) bool {
	all := append([]string{"META-INF/", ".sha1", ".md5", ".txt"}, excludes...)
	for _, exclude := range all {
		exclude = strings.TrimPrefix(filepath.ToSlash(exclude), "./")
		if strings.HasSuffix(exclude, "/") {
			if strings.HasPrefix(name, exclude) {
				return true
			}
			continue
		}
		if name == exclude || strings.HasPrefix(name, exclude+"/") {
			return true
		}
	}
	return false
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

	_, err = io.Copy(outFile, io.LimitReader(rc, maxNativeFileBytes+1))
	return err
}
