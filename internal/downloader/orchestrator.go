package downloader

import (
	"context"
	"fmt"
	"net/http"
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

	o.pool.Wait()

	if errs := o.pool.Errors(); len(errs) > 0 {
		return fmt.Errorf("download failed: %d errors: %w", len(errs), errs[0])
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
