package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	manifestURL     = "https://piston-meta.mojang.com/mc/game/version_manifest_v2.json"
	cacheTTL        = 30 * time.Minute
)

// Client fetches and caches Mojang version metadata.
type Client struct {
	cacheDir  string
	httpClient *http.Client
}

// NewClient creates a metadata client with disk cache under dataRoot/versions/.
func NewClient(dataRoot string) *Client {
	cacheDir := filepath.Join(dataRoot, "versions")
	os.MkdirAll(cacheDir, 0o755)
	return &Client{
		cacheDir: cacheDir,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// FetchManifest returns the version manifest, using disk cache when fresh.
func (c *Client) FetchManifest(ctx context.Context) (*VersionManifest, error) {
	cachePath := filepath.Join(c.cacheDir, "manifest.json")

	// Try disk cache
	if data, err := os.ReadFile(cachePath); err == nil {
		var manifest VersionManifest
		if err := json.Unmarshal(data, &manifest); err == nil {
			return &manifest, nil
		}
	}

	// Fetch fresh
	data, err := c.fetchURL(ctx, manifestURL)
	if err != nil {
		return nil, fmt.Errorf("fetch manifest: %w", err)
	}

	// Write cache
	os.WriteFile(cachePath, data, 0o644)

	var manifest VersionManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("unmarshal manifest: %w", err)
	}
	return &manifest, nil
}

// FetchVersionDetail returns version metadata, using disk cache when fresh.
// It does NOT resolve inheritsFrom — use ResolveVersionChain for that.
func (c *Client) FetchVersionDetail(ctx context.Context, id string) (*VersionDetail, error) {
	cachePath := filepath.Join(c.cacheDir, id+".json")

	// Try disk cache
	if data, err := os.ReadFile(cachePath); err == nil {
		var detail VersionDetail
		if err := json.Unmarshal(data, &detail); err == nil {
			return &detail, nil
		}
	}

	// Need manifest to find version URL
	manifest, err := c.FetchManifest(ctx)
	if err != nil {
		return nil, err
	}

	// Find version entry
	var entry *VersionEntry
	for i := range manifest.Versions {
		if manifest.Versions[i].ID == id {
			entry = &manifest.Versions[i]
			break
		}
	}
	if entry == nil {
		return nil, fmt.Errorf("version %q not found in manifest", id)
	}

	// Fetch version detail
	data, err := c.fetchURL(ctx, entry.URL)
	if err != nil {
		return nil, fmt.Errorf("fetch version %s: %w", id, err)
	}

	// Write cache
	os.WriteFile(cachePath, data, 0o644)

	var detail VersionDetail
	if err := json.Unmarshal(data, &detail); err != nil {
		return nil, fmt.Errorf("unmarshal version %s: %w", id, err)
	}
	return &detail, nil
}

// ResolveVersionChain fetches a version and resolves its full inheritsFrom chain.
func (c *Client) ResolveVersionChain(ctx context.Context, id string) (*VersionDetail, error) {
	return c.resolveChain(ctx, id, make(map[string]bool))
}

func (c *Client) resolveChain(ctx context.Context, id string, visited map[string]bool) (*VersionDetail, error) {
	if visited[id] {
		return nil, fmt.Errorf("inheritsFrom loop detected at %q", id)
	}
	visited[id] = true

	detail, err := c.FetchVersionDetail(ctx, id)
	if err != nil {
		return nil, err
	}

	if detail.InheritsFrom == "" {
		return detail, nil
	}

	parent, err := c.resolveChain(ctx, detail.InheritsFrom, visited)
	if err != nil {
		return nil, fmt.Errorf("resolve parent %s: %w", detail.InheritsFrom, err)
	}

	return MergeVersions(detail, parent), nil
}

func (c *Client) fetchURL(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, url)
	}
	var buf json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&buf); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return buf, nil
}
