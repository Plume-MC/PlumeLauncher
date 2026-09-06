package downloader

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"plumelauncher/internal/metadata"
	"plumelauncher/internal/security"
)

// Orchestrator downloads artifacts from an ArtifactPlan.
type Orchestrator struct {
	dataRoot string
	workers  int
	pool     *Pool
	progress *ProgressTracker
	client   *http.Client
}

// NewOrchestrator creates a download orchestrator for the given data root.
func NewOrchestrator(dataRoot string, workers int) *Orchestrator {
	return &Orchestrator{
		dataRoot: dataRoot,
		workers:  workers,
		client:   &http.Client{Timeout: 30 * time.Second},
	}
}

// DownloadPlan downloads and verifies all artifacts in the plan before returning.
func (o *Orchestrator) DownloadPlan(ctx context.Context, plan *metadata.ArtifactPlan) error {
	o.progress = NewProgressTracker(len(plan.Artifacts))

	// Calculate total size
	var totalSize int64
	for _, a := range plan.Artifacts {
		totalSize += a.Size
	}
	o.progress.AddTotal(0, totalSize)

	tasks := make([]Task, 0, len(plan.Artifacts))
	for _, artifact := range plan.Artifacts {
		fullPath, err := security.ResolveUnderRoot(o.dataRoot, artifact.Path)
		if err != nil {
			return err
		}
		a := artifact
		tasks = append(tasks, Task{
			URL:  a.URL,
			Path: fullPath,
			SHA1: a.Sha1,
			Size: a.Size,
			OnComplete: func(cached bool) {
				if cached {
					o.progress.Increment(a.Size, a.Size)
				} else {
					o.progress.Complete()
				}
			},
			OnProgress: o.progress.Advance,
		})
	}

	return o.downloadTasks(ctx, tasks)
}

// WaitForDownloads waits for all submitted tasks to complete.
func (o *Orchestrator) WaitForDownloads() error {
	if o.pool == nil {
		return nil
	}
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
	if err := security.ValidateArtifactURL(detail.AssetIndex.URL); err != nil {
		return err
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
		resp, err := o.client.Do(req)
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
		if err := os.MkdirAll(filepath.Dir(indexPath), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(indexPath, indexData, 0o644); err != nil {
			return err
		}
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

	// Build asset object downloads
	baseURL := "https://resources.download.minecraft.net/"
	objectsDir := filepath.Join(o.dataRoot, "assets", "objects")

	var tasks []Task
	for _, obj := range indexObj.Objects {
		if len(obj.Hash) != 40 || !isHexHash(obj.Hash) {
			return fmt.Errorf("invalid asset hash %q", obj.Hash)
		}
		p := filepath.Join(objectsDir, obj.Hash[:2], obj.Hash)
		u := baseURL + obj.Hash[:2] + "/" + obj.Hash
		sz := int64(obj.Size)
		tasks = append(tasks, Task{
			URL:  u,
			Path: p,
			SHA1: obj.Hash,
			Size: sz,
			OnComplete: func(cached bool) {
				if cached {
					o.progress.Increment(sz, sz)
				} else {
					o.progress.Complete()
				}
			},
			OnProgress: o.progress.Advance,
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

	return o.downloadTasks(ctx, tasks)
}

func isHexHash(value string) bool {
	for _, char := range value {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') && (char < 'A' || char > 'F') {
			return false
		}
	}
	return true
}

func (o *Orchestrator) downloadTasks(ctx context.Context, tasks []Task) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	o.pool = NewPool(o.workers, o.client)
	o.pool.Start(ctx, filepath.Join(o.dataRoot, "cache"))
	for _, task := range tasks {
		if !o.pool.Submit(ctx, task) {
			waitErr := o.WaitForDownloads()
			if err := ctx.Err(); err != nil {
				return err
			}
			if waitErr != nil {
				return waitErr
			}
			return fmt.Errorf("pool closed, cannot submit task")
		}
	}
	if err := o.WaitForDownloads(); err != nil {
		return err
	}
	return ctx.Err()
}

// Progress returns the current download progress.
func (o *Orchestrator) Progress() ProgressSnapshot {
	if o.progress == nil {
		return ProgressSnapshot{}
	}
	return o.progress.Snapshot()
}
