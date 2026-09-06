package downloader_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"plumelauncher/internal/downloader"
)

func TestPoolBasicDownload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "5")
		w.Write([]byte("hello"))
	}))
	defer server.Close()

	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "cache")
	os.MkdirAll(cacheDir, 0o755)

	pool := downloader.NewPool(2, &http.Client{})
	pool.Start(context.Background(), cacheDir)

	pool.Submit(context.Background(), downloader.Task{
		URL:  server.URL + "/file.txt",
		Path: filepath.Join(dir, "file.txt"),
		Size: 5,
	})

	pool.Wait()

	if len(pool.Errors()) != 0 {
		t.Fatalf("errors: %v", pool.Errors())
	}

	data, _ := os.ReadFile(filepath.Join(dir, "file.txt"))
	if string(data) != "hello" {
		t.Errorf("content = %q, want %q", string(data), "hello")
	}
}

func TestPoolConcurrentDownloads(t *testing.T) {
	var count atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count.Add(1)
		w.Header().Set("Content-Length", "4")
		w.Write([]byte("data"))
	}))
	defer server.Close()

	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "cache")
	os.MkdirAll(cacheDir, 0o755)

	pool := downloader.NewPool(5, &http.Client{})
	pool.Start(context.Background(), cacheDir)

	for i := 0; i < 10; i++ {
		pool.Submit(context.Background(), downloader.Task{
			URL:  server.URL + "/file",
			Path: filepath.Join(dir, "file"+string(rune('0'+i))),
			Size: 4,
		})
	}

	pool.Wait()

	if len(pool.Errors()) != 0 {
		t.Fatalf("errors: %v", pool.Errors())
	}
	if count.Load() != 10 {
		t.Errorf("server hit %d times, want 10", count.Load())
	}
}

func TestPoolSkipsValidFile(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Length", "5")
		w.Write([]byte("hello"))
	}))
	defer server.Close()

	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "cache")
	os.MkdirAll(cacheDir, 0o755)

	finalPath := filepath.Join(dir, "file.txt")
	os.WriteFile(finalPath, []byte("hello"), 0o644)

	pool := downloader.NewPool(1, &http.Client{})
	pool.Start(context.Background(), cacheDir)

	pool.Submit(context.Background(), downloader.Task{
		URL:  server.URL + "/file.txt",
		Path: finalPath,
		SHA1: "aaf4c61ddcc5e8a2dabede0f3b482cd9aea9434d",
		Size: 5,
	})

	pool.Wait()

	if hits.Load() != 0 {
		t.Errorf("server hit %d times, want 0 (file should be skipped)", hits.Load())
	}
	if len(pool.Errors()) != 0 {
		t.Fatalf("errors: %v", pool.Errors())
	}
}

func TestPoolOnComplete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "4")
		w.Write([]byte("done"))
	}))
	defer server.Close()

	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "cache")
	os.MkdirAll(cacheDir, 0o755)

	var called atomic.Bool
	pool := downloader.NewPool(1, &http.Client{})
	pool.Start(context.Background(), cacheDir)

	pool.Submit(context.Background(), downloader.Task{
		URL:  server.URL + "/file",
		Path: filepath.Join(dir, "file.txt"),
		Size: 4,
		OnComplete: func(bool) {
			called.Store(true)
		},
	})

	pool.Wait()

	if !called.Load() {
		t.Error("OnComplete was not called")
	}
}

func TestPoolSubmitAfterWait(t *testing.T) {
	pool := downloader.NewPool(1, &http.Client{})
	pool.Start(context.Background(), "")
	pool.Wait()

	// Submit after wait should return false
	ok := pool.Submit(context.Background(), downloader.Task{URL: "http://example.com", Size: 1})
	if ok {
		t.Error("Submit after Wait should return false")
	}
}

func TestPoolSubmitWaitsForCapacity(t *testing.T) {
	pool := downloader.NewPool(1, &http.Client{})
	ctx := context.Background()
	if !pool.Submit(ctx, downloader.Task{Size: 1}) {
		t.Fatal("first Submit returned false")
	}

	submitted := make(chan bool, 1)
	go func() {
		submitted <- pool.Submit(ctx, downloader.Task{Size: 1})
	}()

	select {
	case <-submitted:
		t.Fatal("Submit returned before a worker had capacity")
	case <-time.After(50 * time.Millisecond):
	}

	pool.Start(ctx, t.TempDir())
	if ok := <-submitted; !ok {
		t.Fatal("blocked Submit returned false")
	}
	pool.Wait()
}
