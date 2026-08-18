package metadata

import "testing"

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
