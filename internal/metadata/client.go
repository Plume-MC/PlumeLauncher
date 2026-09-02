package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"plumelauncher/internal/security"
)

const (
	manifestURL = "https://piston-meta.mojang.com/mc/game/version_manifest_v2.json"
)

// Client fetches and caches Mojang version metadata.
type Client struct {
	dataRoot   string
	cacheDir   string
	httpClient *http.Client
}

// NewClient creates a metadata client with disk cache under dataRoot/versions/.
func NewClient(dataRoot string) *Client {
	cacheDir := filepath.Join(dataRoot, "versions")
	os.MkdirAll(cacheDir, 0o755)
	return &Client{
		dataRoot:   dataRoot,
		cacheDir:   cacheDir,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// FetchAssetObjects loads and caches the content-addressed asset index.
func (c *Client) FetchAssetObjects(ctx context.Context, index AssetIndex) (map[string]AssetObject, error) {
	if index.URL == "" {
		return nil, nil
	}
	if err := security.ValidateArtifactURL(index.URL); err != nil {
		return nil, err
	}
	cachePath := filepath.Join(c.dataRoot, "assets", "indexes", index.ID+".json")
	data, err := os.ReadFile(cachePath)
	if err != nil {
		data, err = c.fetchURL(ctx, index.URL)
		if err != nil {
			return nil, fmt.Errorf("fetch asset index: %w", err)
		}
		if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(cachePath, data, 0o644); err != nil {
			return nil, err
		}
	}
	var payload struct {
		Objects map[string]AssetObject `json:"objects"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("unmarshal asset index: %w", err)
	}
	for name, object := range payload.Objects {
		if len(object.Hash) != 40 || !isHexHash(object.Hash) || object.Size < 0 {
			return nil, fmt.Errorf("invalid asset object %q", name)
		}
	}
	return payload.Objects, nil
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
	if id == "" || strings.ContainsAny(id, `/\`) || filepath.Base(id) != id {
		return nil, fmt.Errorf("invalid version id %q", id)
	}
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

func (c *Client) fetchText(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, url)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
