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

func TestLoaderMetadataRejectsUnsupportedLoader(t *testing.T) {
	if _, err := NewClient(t.TempDir()).SupportsLoaderVersion(t.Context(), "forge", "1.20.1"); err == nil {
		t.Fatal("expected unsupported loader error")
	}
}
