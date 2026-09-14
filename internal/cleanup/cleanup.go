package cleanup

import (
	"os"
	"path/filepath"
	"time"
)

const (
	maxPartFileAge = 1 * time.Hour
)

// CleanupOrphanedPartFiles removes .part files in cache/java/ that are older
// than the age threshold. These are typically left behind by interrupted downloads.
func CleanupOrphanedPartFiles(dataRoot string) int {
	cacheDir := filepath.Join(dataRoot, "cache", "java")
	return cleanupPartFiles(cacheDir, maxPartFileAge)
}

func cleanupPartFiles(dir string, maxAge time.Duration) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}

	cutoff := time.Now().Add(-maxAge)
	removed := 0

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".part" {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		// Use a small tolerance to avoid edge cases at the exact boundary.
		if info.ModTime().Add(time.Second).Before(cutoff) {
			_ = os.Remove(filepath.Join(dir, entry.Name()))
			removed++
		}
	}

	return removed
}
