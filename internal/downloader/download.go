package downloader

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// DownloadWithResume downloads a URL to partPath with HTTP Range resume support.
// Returns the number of bytes written. On failure, the .part file is preserved for resume.
func DownloadWithResume(ctx context.Context, client *http.Client, url string, partPath string) (int64, error) {
	const maxAttempts = 2

	var offset int64

	// Check existing .part file for resume
	if info, err := os.Stat(partPath); err == nil {
		offset = info.Size()
	}

	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		written, err := doDownload(ctx, client, url, partPath, offset)
		if err == nil {
			return written + offset, nil
		}
		lastErr = err

		// If server returned 416 (Range Not Satisfiable), reset and retry
		if err == errRangeNotSatisfiable {
			os.Remove(partPath)
			offset = 0
			continue
		}

		// For other errors, update offset for resume on retry
		if info, statErr := os.Stat(partPath); statErr == nil {
			offset = info.Size()
		}
	}

	return 0, fmt.Errorf("download failed after %d attempts: %w", maxAttempts, lastErr)
}

var errRangeNotSatisfiable = fmt.Errorf("416 Range Not Satisfiable")

func doDownload(ctx context.Context, client *http.Client, url string, partPath string, offset int64) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}

	if offset > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// Full response — truncate .part and start from scratch
		offset = 0
		os.Remove(partPath)
	case http.StatusPartialContent:
		if !validContentRange(resp.Header.Get("Content-Range"), offset) {
			return 0, fmt.Errorf("invalid Content-Range for offset %d", offset)
		}
	case http.StatusRequestedRangeNotSatisfiable:
		return 0, errRangeNotSatisfiable
	default:
		return 0, fmt.Errorf("HTTP %d: %s", resp.StatusCode, url)
	}

	// Open .part file for writing (create or append)
	os.MkdirAll(filepath.Dir(partPath), 0o755)
	flags := os.O_CREATE | os.O_WRONLY
	if offset == 0 {
		flags |= os.O_TRUNC
	} else {
		flags |= os.O_APPEND
	}

	f, err := os.OpenFile(partPath, flags, 0o644)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	written, err := io.Copy(f, resp.Body)
	return written, err
}

func validContentRange(value string, offset int64) bool {
	if !strings.HasPrefix(value, "bytes ") {
		return false
	}

	parts := strings.Split(strings.TrimPrefix(value, "bytes "), "/")
	if len(parts) != 2 {
		return false
	}

	rangeParts := strings.Split(parts[0], "-")
	if len(rangeParts) != 2 {
		return false
	}

	start, err := strconv.ParseInt(rangeParts[0], 10, 64)
	return err == nil && start == offset
}
