package downloader

import (
	"context"
	"fmt"
	"net/http"
	"sync"
)

// Task is a single download unit.
type Task struct {
	URL      string
	Path     string
	SHA1     string
	Size     int64
	OnComplete func() // called after successful commit
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
		tasks:   make(chan Task, 10000),
		client:  client,
	}
}

// Submit adds a task to the pool. Returns false if the pool is shut down.
func (p *Pool) Submit(task Task) bool {
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
	default:
		p.wg.Done()
		return false
	}
}

// Start launches worker goroutines. Call Submit before Start.
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
		p.process(ctx, task, cacheDir)
		p.wg.Done()
	}
}

func (p *Pool) process(ctx context.Context, task Task, cacheDir string) {
	// Check if file already exists and hash matches
	if task.SHA1 != "" {
		ok, err := VerifyFileSHA1(task.Path, task.SHA1)
		if err == nil && ok {
			if task.OnComplete != nil {
				task.OnComplete()
			}
			return
		}
	}

	partPath := task.Path + ".part"

	written, err := DownloadWithResume(ctx, p.client, task.URL, partPath)
	if err != nil {
		p.errorsMu.Lock()
		p.errors = append(p.errors, fmt.Errorf("download %s: %w", task.URL, err))
		p.errorsMu.Unlock()
		return
	}
	_ = written

	if err := CommitFile(partPath, task.Path, task.SHA1); err != nil {
		p.errorsMu.Lock()
		p.errors = append(p.errors, fmt.Errorf("commit %s: %w", task.Path, err))
		p.errorsMu.Unlock()
		return
	}

	if task.OnComplete != nil {
		task.OnComplete()
	}
}
