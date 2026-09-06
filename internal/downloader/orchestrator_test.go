package downloader_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"plumelauncher/internal/downloader"
	"plumelauncher/internal/metadata"
)

func TestOrchestratorDownloadPlan(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "4")
		w.Write([]byte("data"))
	}))
	defer server.Close()

	dir := t.TempDir()
	orch := downloader.NewOrchestrator(dir, 2)

	plan := &metadata.ArtifactPlan{
		VersionID: "test",
		Artifacts: []metadata.Artifact{
			{
				Role:     metadata.RoleClient,
				URL:      server.URL + "/client.jar",
				Path:     "versions/test/test.jar",
				Size:     4,
				Required: true,
			},
			{
				Role:     metadata.RoleLibrary,
				URL:      server.URL + "/lib.jar",
				Path:     "libraries/com/example/lib/1.0/lib-1.0.jar",
				Size:     4,
				Required: true,
			},
		},
	}

	err := orch.DownloadPlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("DownloadPlan: %v", err)
	}

	// Verify files exist
	if _, err := os.Stat(filepath.Join(dir, "versions", "test", "test.jar")); os.IsNotExist(err) {
		t.Error("client jar not downloaded")
	}
	if _, err := os.Stat(filepath.Join(dir, "libraries", "com", "example", "lib", "1.0", "lib-1.0.jar")); os.IsNotExist(err) {
		t.Error("library jar not downloaded")
	}
}

func TestOrchestratorProgress(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "4")
		w.Write([]byte("data"))
	}))
	defer server.Close()

	dir := t.TempDir()
	orch := downloader.NewOrchestrator(dir, 2)

	plan := &metadata.ArtifactPlan{
		VersionID: "test",
		Artifacts: []metadata.Artifact{
			{URL: server.URL + "/a.jar", Path: "a.jar", Size: 4, Required: true},
			{URL: server.URL + "/b.jar", Path: "b.jar", Size: 4, Required: true},
		},
	}

	err := orch.DownloadPlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("DownloadPlan: %v", err)
	}

	snap := orch.Progress()
	if snap.CompletedFiles != 2 {
		t.Errorf("CompletedFiles = %d, want 2", snap.CompletedFiles)
	}
	if snap.TotalFiles != 2 {
		t.Errorf("TotalFiles = %d, want 2", snap.TotalFiles)
	}
}

func TestOrchestratorCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1000")
		w.Write([]byte("partial"))
	}))
	defer server.Close()

	dir := t.TempDir()
	orch := downloader.NewOrchestrator(dir, 1)

	plan := &metadata.ArtifactPlan{
		VersionID: "test",
		Artifacts: []metadata.Artifact{
			{URL: server.URL + "/big.jar", Path: "big.jar", Size: 1000, Required: true},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	err := orch.DownloadPlan(ctx, plan)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context cancellation", err)
	}
}
