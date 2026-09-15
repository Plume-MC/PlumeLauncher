package metadata

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	fabricLoaderURL = "https://meta.fabricmc.net/v2/versions/loader/"
	quiltLoaderURL  = "https://meta.quiltmc.org/v3/versions/loader/"
	fabricGameURL   = "https://meta.fabricmc.net/v2/versions/game"
	quiltGameURL    = "https://meta.quiltmc.org/v3/versions/game"
	fabricMavenURL  = "https://maven.fabricmc.net/"
	quiltMavenURL   = "https://maven.quiltmc.org/repository/release/"
)

type loaderMetadata struct {
	Loader       loaderArtifact     `json:"loader"`
	Intermediary loaderArtifact     `json:"intermediary"`
	Hashed       loaderArtifact     `json:"hashed"`
	LauncherMeta loaderLauncherMeta `json:"launcherMeta"`
}

type loaderGameVersion struct {
	Version string `json:"version"`
	Stable  bool   `json:"stable"`
}

type loaderArtifact struct {
	Maven    string `json:"maven"`
	Version  string `json:"version"`
	Stable   bool   `json:"stable"`
	FileSize int64  `json:"file_size"`
	Hashes   struct {
		SHA1 string `json:"sha1"`
	} `json:"hashes"`
}

type loaderLauncherMeta struct {
	Libraries map[string][]loaderLibrary `json:"libraries"`
	MainClass loaderMainClass            `json:"mainClass"`
}

type loaderMainClass struct{ Client string }

func (m *loaderMainClass) UnmarshalJSON(data []byte) error {
	var client string
	if err := json.Unmarshal(data, &client); err == nil {
		m.Client = client
		return nil
	}
	var classes struct {
		Client string `json:"client"`
	}
	if err := json.Unmarshal(data, &classes); err != nil {
		return err
	}
	m.Client = classes.Client
	return nil
}

type loaderLibrary struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Size int64  `json:"size"`
	SHA1 string `json:"sha1"`
}

// ResolveFabric merges the current Fabric launcher metadata with Mojang metadata.
func (c *Client) ResolveFabric(ctx context.Context, gameVersion, loaderVersion string) (*VersionDetail, error) {
	return c.resolveLoader(ctx, "fabric", gameVersion, loaderVersion, fabricLoaderURL, fabricMavenURL)
}

// ResolveQuilt merges the current Quilt launcher metadata with Mojang metadata.
func (c *Client) ResolveQuilt(ctx context.Context, gameVersion, loaderVersion string) (*VersionDetail, error) {
	return c.resolveLoader(ctx, "quilt", gameVersion, loaderVersion, quiltLoaderURL, quiltMavenURL)
}

// LoaderVersions returns the loader versions available for a Minecraft release.
func (c *Client) LoaderVersions(ctx context.Context, loader, gameVersion string) ([]string, error) {
	var endpoint string
	switch loader {
	case "fabric":
		endpoint = fabricLoaderURL
	case "quilt":
		endpoint = quiltLoaderURL
	default:
		return nil, fmt.Errorf("unsupported loader %q", loader)
	}
	metadata, err := c.loaderMetadata(ctx, loader, gameVersion, endpoint)
	if err != nil {
		return nil, err
	}
	versions := make([]string, 0, len(metadata))
	seen := make(map[string]struct{}, len(metadata))
	for _, source := range metadata {
		if source.Loader.Version == "" {
			continue
		}
		if _, ok := seen[source.Loader.Version]; !ok {
			versions = append(versions, source.Loader.Version)
			seen[source.Loader.Version] = struct{}{}
		}
	}
	return versions, nil
}

// SupportedLoaderVersions returns stable Minecraft releases published by the loader itself.
func (c *Client) SupportedLoaderVersions(ctx context.Context, loader string) ([]string, error) {
	switch loader {
	case "fabric":
		return c.loaderGameVersions(ctx, loader, fabricGameURL)
	case "quilt":
		return c.loaderGameVersions(ctx, loader, quiltGameURL)
	default:
		return nil, fmt.Errorf("unsupported loader %q", loader)
	}
}

func (c *Client) loaderGameVersions(ctx context.Context, loader, endpoint string) ([]string, error) {
	cachePath := filepath.Join(c.cacheDir, loader+"-game-versions.json")
	data, err := os.ReadFile(cachePath)
	if err != nil {
		data, err = c.fetchURL(ctx, endpoint)
		if err != nil {
			return nil, fmt.Errorf("fetch %s game versions: %w", loader, err)
		}
		_ = os.WriteFile(cachePath, data, 0o644)
	}
	var games []loaderGameVersion
	if err := json.Unmarshal(data, &games); err != nil {
		return nil, fmt.Errorf("unmarshal %s game versions: %w", loader, err)
	}
	versions := make([]string, 0, len(games))
	for _, game := range games {
		if game.Stable {
			versions = append(versions, game.Version)
		}
	}
	return versions, nil
}

func (c *Client) resolveLoader(ctx context.Context, loader, gameVersion, requestedVersion, endpoint, mavenURL string) (*VersionDetail, error) {
	versions, err := c.loaderMetadata(ctx, loader, gameVersion, endpoint)
	if err != nil {
		return nil, err
	}
	base, err := c.ResolveVersionChain(ctx, gameVersion)
	if err != nil {
		return nil, err
	}
	source := stableLoaderVersion(versions)
	if requestedVersion != "" {
		found := false
		for _, version := range versions {
			if version.Loader.Version == requestedVersion {
				source = version
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("%s loader %s is unavailable for Minecraft %s", loader, requestedVersion, gameVersion)
		}
	}
	detail := loaderVersion(loader, gameVersion, source, mavenURL)
	if err := c.backfillLoaderSHA1(ctx, detail); err != nil {
		return nil, err
	}
	return MergeVersions(detail, base), nil
}

// backfillLoaderSHA1 resolves Maven .sha1 sidecars for every loader library,
// including launcherMeta libraries that ship without integrity metadata
// (e.g. Quilt). Fail-fast: a missing sidecar aborts resolution.
func (c *Client) backfillLoaderSHA1(ctx context.Context, detail *VersionDetail) error {
	for i := range detail.Libraries {
		if err := c.resolveArtifactSHA1(ctx, &detail.Libraries[i]); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) resolveArtifactSHA1(ctx context.Context, library *Library) error {
	if library.Downloads == nil || library.Downloads.Artifact.URL == "" {
		return nil
	}
	url := library.Downloads.Artifact.URL + ".sha1"
	sha1, err := c.fetchText(ctx, url)
	if err != nil {
		return fmt.Errorf("fetch %s checksum %s: %w", library.Name, url, err)
	}
	sha1 = strings.TrimSpace(sha1)
	if len(sha1) != 40 {
		return fmt.Errorf("invalid SHA-1 for %s (%s)", library.Name, url)
	}
	if _, err := hex.DecodeString(sha1); err != nil {
		return fmt.Errorf("invalid SHA-1 for %s (%s)", library.Name, url)
	}
	library.Downloads.Artifact.SHA1 = sha1
	return nil
}

func stableLoaderVersion(versions []loaderMetadata) loaderMetadata {
	for _, version := range versions {
		if version.Loader.Stable {
			return version
		}
	}
	return versions[0]
}

// SupportsLoaderVersion reports whether the official loader metadata lists a game version.
func (c *Client) SupportsLoaderVersion(ctx context.Context, loader, gameVersion string) (bool, error) {
	switch loader {
	case "fabric":
		_, err := c.loaderMetadata(ctx, loader, gameVersion, fabricLoaderURL)
		return err == nil, err
	case "quilt":
		_, err := c.loaderMetadata(ctx, loader, gameVersion, quiltLoaderURL)
		return err == nil, err
	default:
		return false, fmt.Errorf("unsupported loader %q", loader)
	}
}

func (c *Client) loaderMetadata(ctx context.Context, loader, gameVersion, endpoint string) ([]loaderMetadata, error) {
	if gameVersion == "" || strings.ContainsAny(gameVersion, `/\\`) {
		return nil, fmt.Errorf("invalid game version %q", gameVersion)
	}
	cachePath := filepath.Join(c.cacheDir, loader+"-"+gameVersion+".json")
	data, err := os.ReadFile(cachePath)
	if err != nil {
		data, err = c.fetchURL(ctx, endpoint+gameVersion)
		if err != nil {
			return nil, fmt.Errorf("fetch %s metadata: %w", loader, err)
		}
		_ = os.WriteFile(cachePath, data, 0o644)
	}
	var versions []loaderMetadata
	if err := json.Unmarshal(data, &versions); err != nil {
		return nil, fmt.Errorf("unmarshal %s metadata: %w", loader, err)
	}
	if len(versions) == 0 {
		return nil, fmt.Errorf("%s does not support Minecraft %s", loader, gameVersion)
	}
	return versions, nil
}

func loaderVersion(loader, gameVersion string, source loaderMetadata, mavenURL string) *VersionDetail {
	version := &VersionDetail{
		ID:        loader + "-loader-" + source.Loader.Version + "-" + gameVersion,
		Jar:       gameVersion,
		MainClass: source.LauncherMeta.MainClass.Client,
		Libraries: []Library{loaderLibraryArtifact(source.Loader, mavenURL), loaderLibraryArtifact(source.Intermediary, fabricMavenURL)},
	}
	if source.Hashed.Maven != "" {
		version.Libraries = append(version.Libraries, loaderLibraryArtifact(source.Hashed, mavenURL))
	}
	for _, scope := range []string{"client", "common"} {
		for _, library := range source.LauncherMeta.Libraries[scope] {
			version.Libraries = append(version.Libraries, loaderLibraryFromMetadata(library))
		}
	}
	return version
}

func loaderLibraryArtifact(source loaderArtifact, mavenURL string) Library {
	return loaderLibraryFromMetadata(loaderLibrary{Name: source.Maven, URL: mavenURL, Size: source.FileSize, SHA1: source.Hashes.SHA1})
}

func loaderLibraryFromMetadata(source loaderLibrary) Library {
	path := ResolveMavenPath(source.Name)
	return Library{
		Name: source.Name,
		Downloads: &LibraryDownloads{Artifact: DownloadInfo{
			URL: strings.TrimRight(source.URL, "/") + "/" + path, Path: path, Size: source.Size, SHA1: source.SHA1,
		}},
	}
}
