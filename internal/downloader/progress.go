package downloader

import (
	"sync"
	"time"
)

// ProgressTracker provides thread-safe download progress tracking.
type ProgressTracker struct {
	mu            sync.Mutex
	totalFiles    int
	totalBytes    int64
	completedFiles int
	completedBytes int64
	networkBytes  int64
	startTime     time.Time
}

// NewProgressTracker creates a tracker for the given number of files.
func NewProgressTracker(totalFiles int) *ProgressTracker {
	return &ProgressTracker{
		totalFiles: totalFiles,
		startTime:  time.Now(),
	}
}

// AddTotal adds to the file and byte counts (for dynamic plan building).
func (p *ProgressTracker) AddTotal(count int, size int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.totalFiles += count
	p.totalBytes += size
}

// Increment records completed bytes for a single file.
func (p *ProgressTracker) Increment(size int64, networkBytes int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.completedFiles++
	p.completedBytes += size
	p.networkBytes += networkBytes
}

// ProgressSnapshot is a point-in-time copy of progress state.
type ProgressSnapshot struct {
	TotalFiles     int
	CompletedFiles int
	TotalBytes     int64
	CompletedBytes int64
	Speed          float64 // bytes per second
	ETA            time.Duration
}

// Snapshot returns a thread-safe copy of current progress.
func (p *ProgressTracker) Snapshot() ProgressSnapshot {
	p.mu.Lock()
	defer p.mu.Unlock()

	elapsed := time.Since(p.startTime).Seconds()
	var speed float64
	if elapsed > 0 {
		speed = float64(p.networkBytes) / elapsed
	}

	var eta time.Duration
	if speed > 0 && p.completedBytes < p.totalBytes {
		remaining := float64(p.totalBytes-p.completedBytes) / speed
		eta = time.Duration(remaining * float64(time.Second))
	}

	return ProgressSnapshot{
		TotalFiles:     p.totalFiles,
		CompletedFiles: p.completedFiles,
		TotalBytes:     p.totalBytes,
		CompletedBytes: p.completedBytes,
		Speed:          speed,
		ETA:            eta,
	}
}
