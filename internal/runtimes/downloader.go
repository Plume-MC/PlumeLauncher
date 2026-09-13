package runtimes

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"plumelauncher/internal/security"
)

// ProgressFunc is called with bytes downloaded during a JDK download.
type ProgressFunc func(bytesRead int64)

// JavaDownloader manages JDK downloads from Adoptium.
type JavaDownloader struct {
	dataRoot   string
	httpClient *http.Client
	mu         sync.Mutex
	active     map[int]*downloadOp
}

type downloadOp struct {
	ctx    context.Context
	cancel context.CancelFunc
}

// NewJavaDownloader creates a downloader that stores JDKs under dataRoot/runtimes/java/.
func NewJavaDownloader(dataRoot string) *JavaDownloader {
	return &JavaDownloader{
		dataRoot:   dataRoot,
		httpClient: &http.Client{Timeout: 0},
		active:     make(map[int]*downloadOp),
	}
}

// Download fetches the latest Temurin JDK for the given major version,
// verifies its SHA-256 checksum, and extracts it to DataRoot/runtimes/java/{major}/.
// Emits progress events via the emitFn callback.
func (d *JavaDownloader) Download(ctx context.Context, major int, emitFn func(string, any)) error {
	ctx, cancel := context.WithCancel(ctx)
	op := &downloadOp{ctx: ctx, cancel: cancel}

	d.mu.Lock()
	if existing, ok := d.active[major]; ok {
		d.mu.Unlock()
		existing.cancel()
		return fmt.Errorf("cancelled previous download for Java %d", major)
	}
	d.active[major] = op
	d.mu.Unlock()

	defer func() {
		d.mu.Lock()
		delete(d.active, major)
		d.mu.Unlock()
	}()

	// 1. Fetch latest release from Adoptium
	release, err := FetchLatestRelease(ctx, major)
	if err != nil {
		d.emitProgress(emitFn, major, "failed", 0, 0, err.Error())
		return fmt.Errorf("fetch adoptium release: %w", err)
	}

	url := release.Binary.Package.Link
	expectedChecksum := release.Binary.Package.Checksum
	totalBytes := release.Binary.Package.Size
	version := release.Version.OpenjdkVersion

	// 2. Prepare directories
	runtimeDir := filepath.Join(d.dataRoot, "runtimes", "java", fmt.Sprintf("%d", major))
	cacheDir := filepath.Join(d.dataRoot, "cache", "java")
	archiveName := release.Binary.Package.Name
	archivePath := filepath.Join(cacheDir, archiveName)
	partPath := archivePath + ".part"

	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		d.emitProgress(emitFn, major, "failed", 0, 0, err.Error())
		return err
	}

	// 3. Download with resume
	d.emitProgress(emitFn, major, "downloading", 0, totalBytes, "")

	var bytesRead int64

	// Check existing .part for resume
	if info, err := os.Stat(partPath); err == nil {
		bytesRead = info.Size()
	}

	// Fetch from Adoptium URL — validate it first
	if err := security.ValidateArtifactURL(url); err != nil {
		d.emitProgress(emitFn, major, "failed", bytesRead, totalBytes, "untrusted download URL")
		return fmt.Errorf("untrusted URL: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		d.emitProgress(emitFn, major, "failed", bytesRead, totalBytes, err.Error())
		return err
	}

	if bytesRead > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", bytesRead))
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		d.emitProgress(emitFn, major, "failed", bytesRead, totalBytes, err.Error())
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// Full response — reset
		bytesRead = 0
		os.Remove(partPath)
	case http.StatusPartialContent:
		// Resume — continue
	case http.StatusRequestedRangeNotSatisfiable:
		// Reset
		bytesRead = 0
		os.Remove(partPath)
	default:
		err := fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
		d.emitProgress(emitFn, major, "failed", bytesRead, totalBytes, err.Error())
		return err
	}

	// Open .part for writing
	flags := os.O_CREATE | os.O_WRONLY
	if bytesRead == 0 {
		flags |= os.O_TRUNC
	} else {
		flags |= os.O_APPEND
	}

	f, err := os.OpenFile(partPath, flags, 0o644)
	if err != nil {
		d.emitProgress(emitFn, major, "failed", bytesRead, totalBytes, err.Error())
		return err
	}

	// Progress tracking
	startTime := time.Now()
	lastReport := startTime
	var lastBytes int64 = bytesRead

	var reader io.ReadCloser = resp.Body
	if totalBytes > 0 {
		reader = &progressReader{
			ReadCloser: resp.Body,
			onProgress: func(n int64) {
				bytesRead += n
				now := time.Now()
				if now.Sub(lastReport) >= 250*time.Millisecond || bytesRead == totalBytes {
					elapsed := now.Sub(startTime).Seconds()
					speed := float64(bytesRead) / elapsed
					var eta float64
					if speed > 0 {
						eta = float64(totalBytes-bytesRead) / speed
					}
					d.emitProgress(emitFn, major, "downloading", bytesRead, totalBytes, "")
					_ = lastBytes
					_ = speed
					_ = eta
					lastReport = now
				}
			},
		}
	}

	written, err := io.Copy(f, reader)
	f.Close()

	if err != nil {
		d.emitProgress(emitFn, major, "failed", bytesRead, totalBytes, err.Error())
		return err
	}

	_ = written

	if ctx.Err() != nil {
		d.emitProgress(emitFn, major, "cancelled", bytesRead, totalBytes, "")
		return ctx.Err()
	}

	// 4. Verify SHA-256 checksum
	d.emitProgress(emitFn, major, "verifying", bytesRead, totalBytes, "")

	if err := verifyChecksum(partPath, expectedChecksum); err != nil {
		os.Remove(partPath)
		d.emitProgress(emitFn, major, "failed", bytesRead, totalBytes, "checksum mismatch")
		return fmt.Errorf("checksum verification failed: %w", err)
	}

	// 5. Extract
	d.emitProgress(emitFn, major, "extracting", totalBytes, totalBytes, "")

	if err := os.RemoveAll(runtimeDir); err != nil {
		d.emitProgress(emitFn, major, "failed", totalBytes, totalBytes, err.Error())
		return err
	}

	if err := ExtractJDK(partPath, runtimeDir); err != nil {
		d.emitProgress(emitFn, major, "failed", totalBytes, totalBytes, err.Error())
		return err
	}

	// 6. Write manifest
	m := manifest{
		Major:       major,
		Version:     version,
		InstalledAt: time.Now(),
	}
	if err := writeManifest(runtimeDir, m); err != nil {
		d.emitProgress(emitFn, major, "failed", totalBytes, totalBytes, err.Error())
		return err
	}

	// 7. Cleanup archive
	os.Remove(partPath)
	os.Remove(archivePath)

	d.emitProgress(emitFn, major, "completed", totalBytes, totalBytes, "")
	return nil
}

// Cancel stops an active download for the given major version.
func (d *JavaDownloader) Cancel(major int) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	op, ok := d.active[major]
	if !ok {
		return fmt.Errorf("no active download for Java %d", major)
	}
	op.cancel()
	return nil
}

// IsDownloading reports whether a download is active for the given major version.
func (d *JavaDownloader) IsDownloading(major int) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, ok := d.active[major]
	return ok
}

func (d *JavaDownloader) emitProgress(emitFn func(string, any), major int, status string, bytesRead, totalBytes int64, errMsg string) {
	if emitFn == nil {
		return
	}
	emitFn("java-download-progress", JavaDownloadProgressEvent{
		Major:      major,
		Status:     status,
		BytesRead:  bytesRead,
		TotalBytes: totalBytes,
		Error:      errMsg,
	})
}

func verifyChecksum(filePath, expected string) error {
	if expected == "" {
		return nil // no checksum to verify
	}

	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}

	actual := hex.EncodeToString(h.Sum(nil))
	if actual != expected {
		return fmt.Errorf("expected %s, got %s", expected, actual)
	}
	return nil
}

type progressReader struct {
	io.ReadCloser
	onProgress func(int64)
}

func (r *progressReader) Read(p []byte) (int, error) {
	n, err := r.ReadCloser.Read(p)
	if n > 0 && r.onProgress != nil {
		r.onProgress(int64(n))
	}
	return n, err
}
