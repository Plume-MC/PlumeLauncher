package downloader

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"plumelauncher/internal/metadata"
)

// Orchestrator downloads artifacts from an ArtifactPlan.
type Orchestrator struct {
	dataRoot string
	pool     *Pool
	progress *ProgressTracker
}

// NewOrchestrator creates a download orchestrator for the given data root.
func NewOrchestrator(dataRoot string, workers int) *Orchestrator {
	return &Orchestrator{
		dataRoot: dataRoot,
		pool:     NewPool(workers, &http.Client{}),
	}
}

// DownloadPlan downloads all artifacts in the plan.
// Does NOT call pool.Wait() — caller should call DownloadAssets or pool.Wait after.
func (o *Orchestrator) DownloadPlan(ctx context.Context, plan *metadata.ArtifactPlan) error {
	o.progress = NewProgressTracker(len(plan.Artifacts))

	// Calculate total size
	var totalSize int64
	for _, a := range plan.Artifacts {
		totalSize += a.Size
	}
	o.progress.AddTotal(0, totalSize)

	o.pool.Start(ctx, filepath.Join(o.dataRoot, "cache"))

	for _, artifact := range plan.Artifacts {
		fullPath := filepath.Join(o.dataRoot, artifact.Path)
		task := Task{
			URL:      artifact.URL,
			Path:     fullPath,
			SHA1:     artifact.Sha1,
			Size:     artifact.Size,
			OnComplete: func() {
				o.progress.Increment(artifact.Size, artifact.Size)
			},
		}

		if !o.pool.Submit(task) {
			return fmt.Errorf("pool closed, cannot submit task")
		}
	}

	return nil
}

// WaitForDownloads waits for all submitted tasks to complete.
func (o *Orchestrator) WaitForDownloads() error {
	o.pool.Wait()

	if errs := o.pool.Errors(); len(errs) > 0 {
		return fmt.Errorf("download failed: %d errors: %w", len(errs), errs[0])
	}
	return nil
}

// DownloadAssets downloads all asset objects referenced by the asset index.
func (o *Orchestrator) DownloadAssets(ctx context.Context, detail metadata.VersionDetail) error {
	if detail.AssetIndex.URL == "" {
		return nil
	}

	// Fetch asset index
	indexPath := filepath.Join(o.dataRoot, "assets", "indexes", detail.AssetIndex.ID+".json")
	var indexData []byte

	// Try cache first
	if cached, err := os.ReadFile(indexPath); err == nil {
		indexData = cached
	} else {
		// Fetch from network
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, detail.AssetIndex.URL, nil)
		if err != nil {
			return err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("fetch asset index: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("asset index HTTP %d", resp.StatusCode)
		}
		indexData, err = io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		// Cache
		os.MkdirAll(filepath.Dir(indexPath), 0o755)
		os.WriteFile(indexPath, indexData, 0o644)
	}

	// Parse asset index
	var indexObj struct {
		Objects map[string]struct {
			Hash string `json:"hash"`
			Size int    `json:"size"`
		} `json:"objects"`
	}
	if err := json.Unmarshal(indexData, &indexObj); err != nil {
		return fmt.Errorf("parse asset index: %w", err)
	}

	// Submit asset object downloads
	baseURL := "https://resources.download.minecraft.net/"
	objectsDir := filepath.Join(o.dataRoot, "assets", "objects")

	var tasks []Task
	for _, obj := range indexObj.Objects {
		path := filepath.Join(objectsDir, obj.Hash[:2], obj.Hash)
		url := baseURL + obj.Hash[:2] + "/" + obj.Hash
		tasks = append(tasks, Task{
			URL:  url,
			Path: path,
			SHA1: obj.Hash,
			Size: int64(obj.Size),
			OnComplete: func() {
				o.progress.Increment(int64(obj.Size), int64(obj.Size))
			},
		})
	}

	if len(tasks) == 0 {
		return nil
	}

	// Update progress tracker
	var totalSize int64
	for _, t := range tasks {
		totalSize += t.Size
	}
	o.progress.AddTotal(len(tasks), totalSize)

	// Submit all tasks
	for _, task := range tasks {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if !o.pool.Submit(task) {
			return fmt.Errorf("pool closed, cannot submit task")
		}
	}

	return nil
}

// Progress returns the current download progress.
func (o *Orchestrator) Progress() ProgressSnapshot {
	if o.progress == nil {
		return ProgressSnapshot{}
	}
	return o.progress.Snapshot()
}
