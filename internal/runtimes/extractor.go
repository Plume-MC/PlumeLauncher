package runtimes

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"plumelauncher/internal/security"
)

const (
	maxJDKEntries    = 8192
	maxJDKFileBytes  = 256 << 20 // 256 MB per file
	maxJDKTotalBytes = 1 << 30   // 1 GB total
)

// ExtractJDK extracts a JDK archive (.tar.gz or .zip) into destDir.
// It rejects path traversal, non-metadata symlinks, and excessively large archives.
func ExtractJDK(archivePath string, destDir string) error {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}

	switch {
	case strings.HasSuffix(archivePath, ".tar.gz") || strings.HasSuffix(archivePath, ".tgz"):
		return extractTarGz(archivePath, destDir)
	case strings.HasSuffix(archivePath, ".zip"):
		return extractZip(archivePath, destDir)
	default:
		return fmt.Errorf("unsupported archive format: %s", archivePath)
	}
}

func extractTarGz(archivePath string, destDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("open gzip: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	var totalBytes uint64
	var count int

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read tar: %w", err)
		}

		count++
		if count > maxJDKEntries {
			return fmt.Errorf("archive has too many entries: %d", count)
		}

		name := filepath.ToSlash(header.Name)

		// Skip the top-level directory entry itself
		if header.Typeflag == tar.TypeDir {
			continue
		}
		stripped := stripTopLevelDir(name)
		if stripped == "" || stripped == "." {
			continue
		}

		// Temurin uses symlinks for duplicate legal notices. They are not needed
		// at runtime, so skip those metadata entries instead of creating links.
		if header.Typeflag == tar.TypeSymlink || os.FileMode(header.Mode)&os.ModeSymlink != 0 {
			if isJDKLegalSymlink(stripped) {
				continue
			}
			return fmt.Errorf("symlink mode not allowed: %s", name)
		}

		// Reject hardlinks
		if header.Typeflag == tar.TypeLink {
			return fmt.Errorf("hardlink entry not allowed: %s", name)
		}

		// Size check
		if uint64(header.Size) > maxJDKFileBytes || totalBytes > maxJDKTotalBytes-uint64(header.Size) {
			return fmt.Errorf("archive exceeds extraction limit: %s", name)
		}
		totalBytes += uint64(header.Size)

		destPath, err := security.ResolveUnderRoot(destDir, stripped)
		if err != nil {
			return fmt.Errorf("path traversal detected: %s", name)
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return err
		}

		mode := os.FileMode(header.Mode) & 0o777
		if mode == 0 {
			mode = 0o644
		}
		outFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
		if err != nil {
			return err
		}

		if _, err := io.Copy(outFile, io.LimitReader(tr, maxJDKFileBytes+1)); err != nil {
			outFile.Close()
			return err
		}
		if err := outFile.Close(); err != nil {
			return err
		}
	}

	return nil
}

func extractZip(archivePath string, destDir string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer r.Close()

	if len(r.File) > maxJDKEntries {
		return fmt.Errorf("archive has too many entries: %d", len(r.File))
	}

	var totalBytes uint64

	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}

		name := filepath.ToSlash(f.Name)
		stripped := stripTopLevelDir(name)
		if stripped == "" || stripped == "." {
			continue
		}

		// See the tar extractor for why legal-document symlinks are skipped.
		if f.Mode()&os.ModeSymlink != 0 {
			if isJDKLegalSymlink(stripped) {
				continue
			}
			return fmt.Errorf("symlink entry not allowed: %s", name)
		}

		// Size check
		if f.UncompressedSize64 > maxJDKFileBytes || totalBytes > maxJDKTotalBytes-f.UncompressedSize64 {
			return fmt.Errorf("archive exceeds extraction limit: %s", name)
		}
		totalBytes += f.UncompressedSize64

		destPath, err := security.ResolveUnderRoot(destDir, stripped)
		if err != nil {
			return fmt.Errorf("path traversal detected: %s", name)
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return err
		}

		outFile, err := os.Create(destPath)
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		if _, err := io.Copy(outFile, io.LimitReader(rc, maxJDKFileBytes+1)); err != nil {
			rc.Close()
			outFile.Close()
			return err
		}
		rc.Close()

		if err := outFile.Close(); err != nil {
			return err
		}
	}

	return nil
}

func isJDKLegalSymlink(name string) bool {
	return name == "legal" || strings.HasPrefix(name, "legal/")
}

// stripTopLevelDir removes the first path component from a slash-separated path.
// "jdk-25.0.4+1/bin/java" → "bin/java"
// "jdk-25.0.4+1/lib/modules" → "lib/modules"
func stripTopLevelDir(name string) string {
	idx := strings.Index(name, "/")
	if idx < 0 {
		return ""
	}
	return name[idx+1:]
}
