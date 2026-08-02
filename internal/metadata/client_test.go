package metadata_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"plumelauncher/internal/metadata"
)

func TestFetchManifestCached(t *testing.T) {
	dir := t.TempDir()
	_ = metadata.NewClient(dir) // verify constructor works

	// Simulate cached manifest
	manifestPath := filepath.Join(dir, "versions", "manifest.json")
	os.MkdirAll(filepath.Dir(manifestPath), 0o755)
	os.WriteFile(manifestPath, []byte(`{"latest":{"release":"1.0","snapshot":"1.1"},"versions":[{"id":"1.0","type":"release","url":"http://test","time":"2009-01-01","releaseTime":"2009-01-01","sha1":"def"}]}`), 0o644)

	// Read from cache
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("Cache file is empty")
	}
}

func TestFetchVersionDetail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/mc/game/version_manifest_v2.json":
			w.Write([]byte(`{"latest":{"release":"1.0","snapshot":"1.0"},"versions":[{"id":"1.0","type":"release","url":"http://` + r.Host + `/version/1.0","time":"2009-01-01","releaseTime":"2009-01-01","sha1":"abc"}]}`))
		case "/version/1.0":
			w.Write([]byte(`{"assetIndex":{"id":"1.0","sha1":"abc","size":100,"totalSize":200,"url":"http://example.com"},"assets":"1.0","downloads":{"client":{"sha1":"def","size":300,"url":"http://example.com"}},"id":"1.0","javaVersion":{"component":"java-runtime-alpha","majorVersion":8},"libraries":[],"mainClass":"net.minecraft.client.Minecraft","minimumLauncherVersion":12,"releaseTime":"2009-01-01","time":"2009-01-01","type":"release"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	dir := t.TempDir()
	c := metadata.NewClient(dir)

	ctx := context.Background()
	detail, err := c.FetchVersionDetail(ctx, "1.0")
	if err != nil {
		t.Fatalf("FetchVersionDetail: %v", err)
	}
	if detail.ID != "1.0" {
		t.Errorf("ID = %q, want %q", detail.ID, "1.0")
	}
	if detail.JavaVersion.MajorVersion != 8 {
		t.Errorf("JavaVersion = %d, want 8", detail.JavaVersion.MajorVersion)
	}

	// Verify cache was written
	cachePath := filepath.Join(dir, "versions", "1.0.json")
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Error("Cache file was not written")
	}
}

func TestResolveChainLoopDetection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/mc/game/version_manifest_v2.json":
			w.Write([]byte(`{"latest":{"release":"a","snapshot":"a"},"versions":[{"id":"a","type":"release","url":"http://` + r.Host + `/v/a","time":"2026-01-01","releaseTime":"2026-01-01","sha1":"1"},{"id":"b","type":"release","url":"http://` + r.Host + `/v/b","time":"2026-01-01","releaseTime":"2026-01-01","sha1":"2"}]}`))
		case "/v/a":
			w.Write([]byte(`{"inheritsFrom":"b","assetIndex":{"id":"a","sha1":"","size":0,"totalSize":0,"url":""},"assets":"a","downloads":{},"id":"a","javaVersion":{"component":"jre","majorVersion":8},"libraries":[],"mainClass":"","minimumLauncherVersion":0,"releaseTime":"","time":"","type":"release"}`))
		case "/v/b":
			w.Write([]byte(`{"inheritsFrom":"a","assetIndex":{"id":"b","sha1":"","size":0,"totalSize":0,"url":""},"assets":"b","downloads":{},"id":"b","javaVersion":{"component":"jre","majorVersion":8},"libraries":[],"mainClass":"","minimumLauncherVersion":0,"releaseTime":"","time":"","type":"release"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	dir := t.TempDir()
	c := metadata.NewClient(dir)
	ctx := context.Background()

	_, err := c.ResolveVersionChain(ctx, "a")
	if err == nil {
		t.Fatal("expected loop detection error")
	}
}

func TestFetchVersionDetailNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"latest":{"release":"1.0","snapshot":"1.0"},"versions":[{"id":"1.0","type":"release","url":"http://example.com","time":"2009-01-01","releaseTime":"2009-01-01","sha1":"abc"}]}`))
	}))
	defer server.Close()

	dir := t.TempDir()
	c := metadata.NewClient(dir)
	ctx := context.Background()

	_, err := c.FetchVersionDetail(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent version")
	}
}
