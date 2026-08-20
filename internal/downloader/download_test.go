package downloader_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"plumelauncher/internal/downloader"
)

func TestDownloadWithResumeFull(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "5")
		w.Write([]byte("hello"))
	}))
	defer server.Close()

	dir := t.TempDir()
	partPath := filepath.Join(dir, "test.txt.part")

	ctx := context.Background()
	client := &http.Client{}
	written, err := downloader.DownloadWithResume(ctx, client, server.URL, partPath)
	if err != nil {
		t.Fatalf("DownloadWithResume: %v", err)
	}
	if written != 5 {
		t.Errorf("written = %d, want 5", written)
	}

	data, _ := os.ReadFile(partPath)
	if string(data) != "hello" {
		t.Errorf("content = %q, want %q", string(data), "hello")
	}
}

func TestDownloadWithResumePartial(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rangeHeader := r.Header.Get("Range")
		if rangeHeader == "bytes=5-" {
			w.Header().Set("Content-Range", "bytes 5-9/10")
			w.Header().Set("Content-Length", "5")
			w.WriteHeader(http.StatusPartialContent)
			w.Write([]byte("world"))
		} else {
			w.Header().Set("Content-Length", "10")
			w.Write([]byte("helloworld"))
		}
	}))
	defer server.Close()

	dir := t.TempDir()
	partPath := filepath.Join(dir, "test.txt.part")
	// Write partial content
	os.WriteFile(partPath, []byte("hello"), 0o644)

	ctx := context.Background()
	client := &http.Client{}
	written, err := downloader.DownloadWithResume(ctx, client, server.URL, partPath)
	if err != nil {
		t.Fatalf("DownloadWithResume: %v", err)
	}
	if written != 10 {
		t.Errorf("written = %d, want 10 (5 existing + 5 new)", written)
	}

	data, _ := os.ReadFile(partPath)
	if string(data) != "helloworld" {
		t.Errorf("content = %q, want %q", string(data), "helloworld")
	}
}

func TestDownloadWithResumeRangeNotSatisfiable(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 && r.Header.Get("Range") != "" {
			w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
			return
		}
		w.Header().Set("Content-Length", "5")
		w.Write([]byte("hello"))
	}))
	defer server.Close()

	dir := t.TempDir()
	partPath := filepath.Join(dir, "test.txt.part")
	os.WriteFile(partPath, []byte("stale"), 0o644)

	ctx := context.Background()
	client := &http.Client{}
	written, err := downloader.DownloadWithResume(ctx, client, server.URL, partPath)
	if err != nil {
		t.Fatalf("DownloadWithResume: %v", err)
	}
	if written != 5 {
		t.Errorf("written = %d, want 5", written)
	}
}

func TestDownloadWithResumeInvalidContentRangeRestarts(t *testing.T) {
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests == 1 {
			w.Header().Set("Content-Range", "bytes 2-4/5")
			w.WriteHeader(http.StatusPartialContent)
			_, _ = w.Write([]byte("bad"))
			return
		}
		w.Header().Set("Content-Length", "5")
		_, _ = w.Write([]byte("hello"))
	}))
	defer server.Close()

	partPath := filepath.Join(t.TempDir(), "test.txt.part")
	if err := os.WriteFile(partPath, []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}
	written, err := downloader.DownloadWithResume(context.Background(), &http.Client{}, server.URL, partPath)
	if err != nil {
		t.Fatalf("DownloadWithResume: %v", err)
	}
	if written != 5 {
		t.Fatalf("written = %d, want 5", written)
	}
	data, _ := os.ReadFile(partPath)
	if string(data) != "hello" {
		t.Fatalf("content = %q, want hello", data)
	}
}

func TestDownloadWithResumeCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1000")
		w.Write([]byte("partial"))
	}))
	defer server.Close()

	dir := t.TempDir()
	partPath := filepath.Join(dir, "test.txt.part")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	client := &http.Client{}
	_, err := downloader.DownloadWithResume(ctx, client, server.URL, partPath)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}
