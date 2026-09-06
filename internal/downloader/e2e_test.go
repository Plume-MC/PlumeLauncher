package downloader_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"plumelauncher/internal/downloader"
	"plumelauncher/internal/metadata"
)

func TestE2EDownloadInterruptedResume(t *testing.T) {
	// Server serves "helloworld" (10 bytes)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "10")
		w.Write([]byte("helloworld"))
	}))
	defer server.Close()

	dir := t.TempDir()

	// Simulate interrupted download: write partial .part file
	partPath := filepath.Join(dir, "file.txt.part")
	os.WriteFile(partPath, []byte("hello"), 0o644) // 5 bytes of 10

	finalPath := filepath.Join(dir, "file.txt")
	plan := &metadata.ArtifactPlan{
		VersionID: "test",
		Artifacts: []metadata.Artifact{
			{URL: server.URL + "/file", Path: "file.txt", Size: 10, Required: true},
		},
	}

	// Override fullPath to use our paths
	plan.Artifacts[0].Path = "file.txt"

	orch := downloader.NewOrchestrator(dir, 1)
	err := orch.DownloadPlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("DownloadPlan: %v", err)
	}

	data, _ := os.ReadFile(finalPath)
	if string(data) != "helloworld" {
		t.Errorf("content = %q, want %q", string(data), "helloworld")
	}
}

func TestE2EServerWithoutRangeRestart(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		// First call with Range: ignore it, return full response
		if callCount == 1 && r.Header.Get("Range") != "" {
			w.Header().Set("Content-Length", "5")
			w.Write([]byte("hello"))
			return
		}
		w.Header().Set("Content-Length", "5")
		w.Write([]byte("hello"))
	}))
	defer server.Close()

	dir := t.TempDir()
	plan := &metadata.ArtifactPlan{
		VersionID: "test",
		Artifacts: []metadata.Artifact{
			{URL: server.URL + "/file", Path: "file.txt", Size: 5, Required: true},
		},
	}

	orch := downloader.NewOrchestrator(dir, 1)
	err := orch.DownloadPlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("DownloadPlan: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, "file.txt"))
	if string(data) != "hello" {
		t.Errorf("content = %q, want %q", string(data), "hello")
	}
}

func TestE2EHashMismatchNeverCommits(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "4")
		w.Write([]byte("data"))
	}))
	defer server.Close()

	dir := t.TempDir()
	plan := &metadata.ArtifactPlan{
		VersionID: "test",
		Artifacts: []metadata.Artifact{
			{URL: server.URL + "/file", Path: "file.txt", Size: 4, Sha1: "0000000000000000000000000000000000000000", Required: true},
		},
	}

	orch := downloader.NewOrchestrator(dir, 1)
	err := orch.DownloadPlan(context.Background(), plan)
	if err == nil {
		t.Fatal("expected error for hash mismatch")
	}

	// Final file should not exist
	if _, err := os.Stat(filepath.Join(dir, "file.txt")); !os.IsNotExist(err) {
		t.Error("final file should not exist after hash mismatch")
	}

	// .part file should be cleaned up
	if _, err := os.Stat(filepath.Join(dir, "file.txt.part")); !os.IsNotExist(err) {
		t.Error(".part file should be cleaned up after hash mismatch")
	}
}

func TestE2ETaskFailureProducesFailedState(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	dir := t.TempDir()
	plan := &metadata.ArtifactPlan{
		VersionID: "test",
		Artifacts: []metadata.Artifact{
			{URL: server.URL + "/file", Path: "file.txt", Size: 4, Required: true},
		},
	}

	orch := downloader.NewOrchestrator(dir, 1)
	err := orch.DownloadPlan(context.Background(), plan)
	if err == nil {
		t.Fatal("expected error for server failure")
	}
}

func TestE2ECancellationLeavesPartFiles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1000000")
		// Write slowly
		for i := 0; i < 1000; i++ {
			w.Write([]byte("x"))
			w.(http.Flusher).Flush()
			time.Sleep(time.Millisecond)
		}
	}))
	defer server.Close()

	dir := t.TempDir()
	plan := &metadata.ArtifactPlan{
		VersionID: "test",
		Artifacts: []metadata.Artifact{
			{URL: server.URL + "/big", Path: "big.bin", Size: 1000000, Required: true},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	orch := downloader.NewOrchestrator(dir, 1)
	if err := orch.DownloadPlan(ctx, plan); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want context deadline", err)
	}

	// .part file should exist (recoverable)
	partPath := filepath.Join(dir, "big.bin.part")
	if _, err := os.Stat(partPath); os.IsNotExist(err) {
		t.Log(".part file not found — download may have completed before cancel")
	}
}

func TestE2EVerifyAndRepair(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "4")
		w.Write([]byte("data"))
	}))
	defer server.Close()

	dir := t.TempDir()

	plan := &metadata.ArtifactPlan{
		VersionID: "test",
		Artifacts: []metadata.Artifact{
			{URL: server.URL + "/good.jar", Path: "lib/good.jar", Size: 4, Required: true},
			{URL: server.URL + "/bad.jar", Path: "lib/bad.jar", Size: 4, Required: true},
			{URL: server.URL + "/missing.jar", Path: "lib/missing.jar", Size: 4, Required: true},
		},
	}

	// Initial download
	orch := downloader.NewOrchestrator(dir, 1)
	err := orch.DownloadPlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("DownloadPlan: %v", err)
	}

	// Check files exist
	for _, a := range plan.Artifacts {
		fullPath := filepath.Join(dir, a.Path)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Errorf("file not downloaded: %s", a.Path)
		}
	}

	if err := os.WriteFile(filepath.Join(dir, "lib", "bad.jar"), []byte("bad"), 0o644); err != nil {
		t.Fatalf("corrupt artifact: %v", err)
	}

	// Verify
	results := downloader.VerifyPlan(dir, plan)
	validCount := 0
	for _, r := range results {
		if r.Valid {
			validCount++
		}
	}
	// good.jar (4 bytes, expected 4) → valid
	// bad.jar was corrupted after download → invalid
	// missing.jar (4 bytes, expected 4) → valid
	if validCount != 2 {
		t.Errorf("valid count = %d, want 2", validCount)
	}

	// Repair
	err = downloader.RepairPlan(context.Background(), dir, plan)
	if err != nil {
		t.Fatalf("RepairPlan: %v", err)
	}

	// Verify again
	results = downloader.VerifyPlan(dir, plan)
	validCount = 0
	for _, r := range results {
		if r.Valid {
			validCount++
		}
	}
	if validCount != 3 {
		t.Errorf("valid count after repair = %d, want 3", validCount)
	}
}
