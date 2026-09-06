package downloader

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"plumelauncher/internal/metadata"
)

// Task is a single download unit.
type Task struct {
	URL        string
	Path       string
	SHA1       string
	Size       int64
	OnComplete func(cached bool) // called after successful commit
	OnProgress func(int64)
}

// Pool is a bounded worker pool for download tasks.
type Pool struct {
	workers  int
	tasks    chan Task
	wg       sync.WaitGroup
	errors   []error
	errorsMu sync.Mutex
	client   *http.Client
	closed   bool
	closedMu sync.Mutex
}

// NewPool creates a bounded worker pool with the given concurrency.
func NewPool(workers int, client *http.Client) *Pool {
	if workers <= 0 {
		workers = 10
	}
	if client == nil {
		client = &http.Client{}
	}
	return &Pool{
		workers: workers,
		tasks:   make(chan Task, workers),
		client:  client,
	}
}

// Submit adds a task to the pool. Returns false if the pool is shut down.
func (p *Pool) Submit(ctx context.Context, task Task) bool {
	p.closedMu.Lock()
	if p.closed {
		p.closedMu.Unlock()
		return false
	}
	p.closedMu.Unlock()

	p.wg.Add(1)
	select {
	case p.tasks <- task:
		return true
	case <-ctx.Done():
		p.wg.Done()
		return false
	}
}

// Start launches worker goroutines before tasks are submitted.
func (p *Pool) Start(ctx context.Context, cacheDir string) {
	for i := 0; i < p.workers; i++ {
		go p.worker(ctx, cacheDir)
	}
}

// Wait blocks until all submitted tasks complete.
func (p *Pool) Wait() {
	p.wg.Wait()
	p.closedMu.Lock()
	p.closed = true
	p.closedMu.Unlock()
	close(p.tasks)
}

// Errors returns all errors collected during execution.
func (p *Pool) Errors() []error {
	p.errorsMu.Lock()
	defer p.errorsMu.Unlock()
	return p.errors
}

func (p *Pool) worker(ctx context.Context, cacheDir string) {
	for task := range p.tasks {
		if ctx.Err() != nil {
			p.wg.Done()
			continue
		}
		p.process(ctx, task, cacheDir)
		p.wg.Done()
	}
}

func (p *Pool) process(ctx context.Context, task Task, cacheDir string) {
	if err := validateIntegrityMetadata(task.SHA1, task.Size); err != nil {
		p.recordError(fmt.Errorf("validate %s: %w", task.Path, err))
		return
	}

	// Reuse an existing artifact only after checking its supplied integrity data.
	status := checkArtifact(task.Path, metadata.Artifact{Path: task.Path, Sha1: task.SHA1, Size: task.Size}, true)
	if status.Valid {
		if task.OnComplete != nil {
			task.OnComplete(true)
		}
		return
	}

	partPath := task.Path + ".part"

	written, err := DownloadWithResumeProgress(ctx, p.client, task.URL, partPath, task.OnProgress)
	if err != nil {
		p.recordError(fmt.Errorf("download %s: %w", task.URL, err))
		return
	}
	if err := ctx.Err(); err != nil {
		p.recordError(err)
		return
	}
	_ = written

	if err := CommitFile(partPath, task.Path, task.SHA1, task.Size); err != nil {
		p.recordError(fmt.Errorf("commit %s: %w", task.Path, err))
		return
	}

	if task.OnComplete != nil {
		task.OnComplete(false)
	}
}

func (p *Pool) recordError(err error) {
	p.errorsMu.Lock()
	p.errors = append(p.errors, err)
	p.errorsMu.Unlock()
}
