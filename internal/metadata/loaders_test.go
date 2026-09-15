package metadata

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoaderMainClassAcceptsFabricAndQuiltShapes(t *testing.T) {
	for _, payload := range []string{`{"mainClass":"net.fabricmc.loader.impl.launch.knot.KnotClient"}`, `{"mainClass":{"client":"org.quiltmc.loader.impl.launch.knot.KnotClient"}}`} {
		var metadata loaderLauncherMeta
		if err := json.Unmarshal([]byte(payload), &metadata); err != nil {
			t.Fatal(err)
		}
		if metadata.MainClass.Client == "" {
			t.Fatal("missing client main class")
		}
	}
}

func TestLoaderGameVersionsOnlyReturnsStableReleases(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"version":"1.21.4","stable":true},{"version":"24w46a","stable":false},{"version":"1.20.4","stable":true}]`))
	}))
	defer server.Close()
	client := NewClient(t.TempDir())
	client.httpClient = server.Client()
	versions, err := client.loaderGameVersions(t.Context(), "fabric", server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 2 || versions[0] != "1.21.4" || versions[1] != "1.20.4" {
		t.Fatalf("versions = %#v", versions)
	}
}

func TestStableLoaderVersionPrefersStableEntry(t *testing.T) {
	version := stableLoaderVersion([]loaderMetadata{{Loader: loaderArtifact{Version: "0.19.2"}}, {Loader: loaderArtifact{Version: "0.19.3", Stable: true}}})
	if version.Loader.Version != "0.19.3" {
		t.Fatalf("version = %q", version.Loader.Version)
	}
}

func TestLoaderVersionsReturnsAvailableLoaderVersions(t *testing.T) {
	client := NewClient(t.TempDir())
	data := []byte(`[{"loader":{"version":"0.19.3","stable":true}},{"loader":{"version":"0.19.2"}}]`)
	if err := os.WriteFile(filepath.Join(client.cacheDir, "fabric-1.21.6.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	versions, err := client.LoaderVersions(t.Context(), "fabric", "1.21.6")
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 2 || versions[0] != "0.19.3" || versions[1] != "0.19.2" {
		t.Fatalf("versions = %#v", versions)
	}
}

func TestLoaderVersionsPrefersLatestStableRelease(t *testing.T) {
	client := NewClient(t.TempDir())
	data := []byte(`[{"loader":{"version":"0.20.0-beta.9"}},{"loader":{"version":"0.30.1-beta.9"}},{"loader":{"version":"0.30.1-beta.10"}},{"loader":{"version":"0.30.2-beta.1"}},{"loader":{"version":"0.30.1"}}]`)
	if err := os.WriteFile(filepath.Join(client.cacheDir, "quilt-26.2.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	versions, err := client.LoaderVersions(t.Context(), "quilt", "26.2")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"0.30.1", "0.30.2-beta.1", "0.30.1-beta.10", "0.30.1-beta.9", "0.20.0-beta.9"}
	if len(versions) != len(want) {
		t.Fatalf("versions = %#v, want %#v", versions, want)
	}
	for i := range want {
		if versions[i] != want[i] {
			t.Fatalf("versions = %#v, want %#v", versions, want)
		}
	}
}

func TestLoaderVersionUsesFabricMavenForQuiltIntermediary(t *testing.T) {
	detail := loaderVersion("quilt", "1.21.11", loaderMetadata{
		Loader:       loaderArtifact{Maven: "org.quiltmc:quilt-loader:0.25.0"},
		Intermediary: loaderArtifact{Maven: "net.fabricmc:intermediary:1.21.11"},
		Hashed:       loaderArtifact{Maven: "org.quiltmc:hashed:1.21.11"},
	}, quiltMavenURL)
	if got, want := detail.Libraries[1].Downloads.Artifact.URL, "https://maven.fabricmc.net/net/fabricmc/intermediary/1.21.11/intermediary-1.21.11.jar"; got != want {
		t.Fatalf("intermediary URL = %q, want %q", got, want)
	}
}

func TestResolveArtifactSHA1UsesMavenSidecar(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/artifact.jar.sha1" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte("56aa2b39b149ce19c6ea86324198e4829e09a4c6\n"))
	}))
	defer server.Close()
	client := NewClient(t.TempDir())
	client.httpClient = server.Client()
	library := Library{Name: "example:artifact:1", Downloads: &LibraryDownloads{Artifact: DownloadInfo{URL: server.URL + "/artifact.jar", SHA1: "stale"}}}
	if err := client.resolveArtifactSHA1(t.Context(), &library); err != nil {
		t.Fatal(err)
	}
	if library.Downloads.Artifact.SHA1 != "56aa2b39b149ce19c6ea86324198e4829e09a4c6" {
		t.Fatalf("SHA1 = %q", library.Downloads.Artifact.SHA1)
	}
}

func TestLoaderVersionUsesAuthoritativeClientLibraries(t *testing.T) {
	source := loaderMetadata{
		Loader:       loaderArtifact{Maven: "net.fabricmc:fabric-loader:0.16.0", Version: "0.16.0"},
		Intermediary: loaderArtifact{Maven: "net.fabricmc:intermediary:1.20.1"},
		LauncherMeta: loaderLauncherMeta{
			Libraries: map[string][]loaderLibrary{
				"client":      {{Name: "example:client:1", URL: "https://repo.example/", Size: 1, SHA1: "client"}},
				"common":      {{Name: "example:common:1", URL: "https://repo.example/", Size: 2, SHA1: "common"}},
				"development": {{Name: "example:dev:1", URL: "https://repo.example/"}},
			},
		},
	}
	detail := loaderVersion("fabric", "1.20.1", source, fabricMavenURL)
	if detail.ID != "fabric-loader-0.16.0-1.20.1" || detail.Jar != "1.20.1" {
		t.Fatalf("unexpected loader identity: %#v", detail)
	}
	if len(detail.Libraries) != 4 {
		t.Fatalf("libraries = %d, want loader, intermediary, client, common", len(detail.Libraries))
	}
	if detail.Libraries[2].Downloads.Artifact.URL != "https://repo.example/example/client/1/client-1.jar" {
		t.Fatal("client library URL was not derived from authoritative metadata")
	}
}

func TestLoaderVersionSkipsMissingIntermediary(t *testing.T) {
	detail := loaderVersion("quilt", "26.2", loaderMetadata{
		Loader: loaderArtifact{Maven: "org.quiltmc:quilt-loader:0.20.0-beta.9", Version: "0.20.0-beta.9"},
	}, quiltMavenURL)

	if len(detail.Libraries) != 1 {
		t.Fatalf("libraries = %d, want only the loader library", len(detail.Libraries))
	}
	if detail.Libraries[0].Name != "org.quiltmc:quilt-loader:0.20.0-beta.9" {
		t.Fatalf("loader library = %q", detail.Libraries[0].Name)
	}
}

func TestLoaderPlanIncludesLoaderArtifacts(t *testing.T) {
	detail := loaderVersion("quilt", "1.20.1", loaderMetadata{
		Loader:       loaderArtifact{Maven: "org.quiltmc:quilt-loader:0.21.0", Version: "0.21.0", FileSize: 12},
		Intermediary: loaderArtifact{Maven: "net.fabricmc:intermediary:1.20.1"},
		Hashed:       loaderArtifact{Maven: "org.quiltmc:hashed:1.20.1", FileSize: 9},
	}, quiltMavenURL)
	plan := ResolvePlan(*detail, SystemInfo{OS: "windows", Arch: "x64"})
	if len(plan.Artifacts) != 3 {
		t.Fatalf("artifacts = %d, want loader, intermediary, hashed", len(plan.Artifacts))
	}
	for _, artifact := range plan.Artifacts {
		if artifact.Role != RoleLibrary || artifact.URL == "" || artifact.Path == "" {
			t.Fatalf("invalid loader artifact: %#v", artifact)
		}
	}
}

type rewriteHostRoundTripper struct {
	host string
}

func (rt rewriteHostRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = "http"
	req.URL.Host = rt.host
	return http.DefaultTransport.RoundTrip(req)
}

func TestBackfillLoaderSHA1CoversLauncherMetaLibraries(t *testing.T) {
	const sidecarSHA1 = "f62a27adbfd8ab4d4fa5681793039f2c0b177155"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, ".jar.sha1") {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(sidecarSHA1 + "\n"))
	}))
	defer server.Close()
	client := NewClient(t.TempDir())
	// Route every sidecar fetch (including hardcoded maven hosts) to the
	// fake server so the test never touches the network.
	client.httpClient = &http.Client{Transport: rewriteHostRoundTripper{host: strings.TrimPrefix(server.URL, "http://")}}
	// Quilt-style metadata: launcherMeta libraries carry no size/hash.
	detail := loaderVersion("quilt", "1.20.1", loaderMetadata{
		Loader:       loaderArtifact{Maven: "org.quiltmc:quilt-loader:0.24.0", Version: "0.24.0"},
		Intermediary: loaderArtifact{Maven: "net.fabricmc:intermediary:1.20.1"},
		Hashed:       loaderArtifact{Maven: "org.quiltmc:hashed:1.20.1"},
		LauncherMeta: loaderLauncherMeta{
			Libraries: map[string][]loaderLibrary{
				"common": {
					{Name: "net.fabricmc:access-widener:2.1.0", URL: server.URL + "/"},
					{Name: "org.quiltmc:quilt-json5:1.0.4", URL: server.URL + "/"},
				},
			},
			MainClass: loaderMainClass{Client: "org.quiltmc.loader.impl.launch.knot.KnotClient"},
		},
	}, server.URL)
	if err := client.backfillLoaderSHA1(t.Context(), detail); err != nil {
		t.Fatal(err)
	}
	for _, library := range detail.Libraries {
		if got := library.Downloads.Artifact.SHA1; got != sidecarSHA1 {
			t.Errorf("library %q SHA1 = %q, want sidecar %q", library.Name, got, sidecarSHA1)
		}
	}
}

func TestLoaderMetadataRejectsUnsupportedLoader(t *testing.T) {
	if _, err := NewClient(t.TempDir()).SupportsLoaderVersion(t.Context(), "forge", "1.20.1"); err == nil {
		t.Fatal("expected unsupported loader error")
	}
}
