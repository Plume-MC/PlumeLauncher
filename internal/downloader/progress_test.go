package downloader_test

import (
	"sync"
	"testing"
	"time"

	"plumelauncher/internal/downloader"
)

func TestProgressTrackerBasic(t *testing.T) {
	tracker := downloader.NewProgressTracker(5)
	tracker.AddTotal(0, 1000)

	snap := tracker.Snapshot()
	if snap.TotalFiles != 5 {
		t.Errorf("TotalFiles = %d, want 5", snap.TotalFiles)
	}
	if snap.TotalBytes != 1000 {
		t.Errorf("TotalBytes = %d, want 1000", snap.TotalBytes)
	}
	if snap.CompletedFiles != 0 {
		t.Errorf("CompletedFiles = %d, want 0", snap.CompletedFiles)
	}
}

func TestProgressTrackerIncrement(t *testing.T) {
	tracker := downloader.NewProgressTracker(2)
	tracker.AddTotal(0, 200)

	tracker.Increment(100, 100)
	tracker.Increment(50, 60)

	snap := tracker.Snapshot()
	if snap.CompletedFiles != 2 {
		t.Errorf("CompletedFiles = %d, want 2", snap.CompletedFiles)
	}
	if snap.CompletedBytes != 150 {
		t.Errorf("CompletedBytes = %d, want 150", snap.CompletedBytes)
	}
}

func TestProgressTrackerSpeed(t *testing.T) {
	tracker := downloader.NewProgressTracker(1)
	tracker.AddTotal(0, 1000)

	// Simulate some progress
	time.Sleep(10 * time.Millisecond)
	tracker.Increment(100, 100)

	snap := tracker.Snapshot()
	if snap.Speed <= 0 {
		t.Errorf("Speed = %f, expected > 0", snap.Speed)
	}
}

func TestProgressTrackerConcurrency(t *testing.T) {
	tracker := downloader.NewProgressTracker(0)
	tracker.AddTotal(10, 10000)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tracker.Increment(100, 100)
		}()
	}
	wg.Wait()

	snap := tracker.Snapshot()
	if snap.CompletedFiles != 10 {
		t.Errorf("CompletedFiles = %d, want 10", snap.CompletedFiles)
	}
	if snap.CompletedBytes != 1000 {
		t.Errorf("CompletedBytes = %d, want 1000", snap.CompletedBytes)
	}
}
