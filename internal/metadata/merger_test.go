package metadata_test

import (
	"testing"

	"plumelauncher/internal/metadata"
)

func TestMergeVersionsFillsMissingAssetIndex(t *testing.T) {
	child := &metadata.VersionDetail{
		ID: "fabric-0.15.0-1.20.1",
	}
	parent := &metadata.VersionDetail{
		ID: "1.20.1",
		AssetIndex: metadata.AssetIndex{
			ID:   "1.20",
			SHA1: "abc",
			Size: 100,
			URL:  "http://example.com/1.20.json",
		},
		Assets: "1.20",
	}

	result := metadata.MergeVersions(child, parent)
	if result.AssetIndex.ID != "1.20" {
		t.Errorf("AssetIndex.ID = %q, want %q", result.AssetIndex.ID, "1.20")
	}
	if result.Assets != "1.20" {
		t.Errorf("Assets = %q, want %q", result.Assets, "1.20")
	}
}

func TestMergeVersionsDeduplicatesLibraries(t *testing.T) {
	child := &metadata.VersionDetail{
		ID: "fabric",
		Libraries: []metadata.Library{
			{Name: "com.mojang:logging:1.0.0"},
			{Name: "net.fabricmc:loader:0.15.0"},
		},
	}
	parent := &metadata.VersionDetail{
		ID: "1.20.1",
		Libraries: []metadata.Library{
			{Name: "com.mojang:logging:1.0.0"}, // duplicate
			{Name: "com.mojang:blocklist:1.0.10"},
		},
	}

	result := metadata.MergeVersions(child, parent)
	if len(result.Libraries) != 3 {
		t.Errorf("Libraries len = %d, want 3", len(result.Libraries))
	}
}

func TestMergeVersionsPreservesNativeLibraryVariant(t *testing.T) {
	child := &metadata.VersionDetail{ID: "fabric"}
	parent := &metadata.VersionDetail{
		ID: "1.16.5",
		Libraries: []metadata.Library{
			{
				Name: "org.lwjgl:lwjgl:3.2.2",
				Downloads: &metadata.LibraryDownloads{Artifact: metadata.DownloadInfo{
					Path: "org/lwjgl/lwjgl/3.2.2/lwjgl-3.2.2.jar",
				}},
			},
			{
				Name:    "org.lwjgl:lwjgl:3.2.2",
				Natives: map[string]string{"linux": "natives-linux"},
				Downloads: &metadata.LibraryDownloads{
					Artifact: metadata.DownloadInfo{Path: "org/lwjgl/lwjgl/3.2.2/lwjgl-3.2.2.jar"},
					Classifiers: map[string]metadata.DownloadInfo{
						"natives-linux": {Path: "org/lwjgl/lwjgl/3.2.2/lwjgl-3.2.2-natives-linux.jar"},
					},
				},
			},
		},
	}

	result := metadata.MergeVersions(child, parent)
	if len(result.Libraries) != 2 {
		t.Fatalf("Libraries len = %d, want 2", len(result.Libraries))
	}
	for _, library := range result.Libraries {
		if path, ok := metadata.ResolveNativePath(library, metadata.SystemInfo{OS: "linux", Arch: "x64"}); ok {
			if path != "org/lwjgl/lwjgl/3.2.2/lwjgl-3.2.2-natives-linux.jar" {
				t.Fatalf("native path = %q", path)
			}
			return
		}
	}
	t.Fatal("native library variant was discarded")
}

func TestMergeVersionsConcatenatesGameArgs(t *testing.T) {
	child := &metadata.VersionDetail{
		ID: "fabric",
		Arguments: &metadata.Arguments{
			Game: []metadata.Argument{{StringValue: "--fabricVersion"}},
		},
	}
	parent := &metadata.VersionDetail{
		ID: "1.20.1",
		Arguments: &metadata.Arguments{
			Game: []metadata.Argument{{StringValue: "--username"}},
		},
	}

	result := metadata.MergeVersions(child, parent)
	if len(result.Arguments.Game) != 2 {
		t.Errorf("Arguments.Game len = %d, want 2", len(result.Arguments.Game))
	}
}

func TestMergeVersionsConcatenatesJvmArgs(t *testing.T) {
	child := &metadata.VersionDetail{
		ID: "fabric",
		Arguments: &metadata.Arguments{
			JVM: []metadata.Argument{{StringValue: "-Dfabric=true"}},
		},
	}
	parent := &metadata.VersionDetail{
		ID: "1.20.1",
		Arguments: &metadata.Arguments{
			JVM: []metadata.Argument{{StringValue: "-Xmx1G"}},
		},
	}

	result := metadata.MergeVersions(child, parent)
	if len(result.Arguments.JVM) != 2 {
		t.Errorf("Arguments.JVM len = %d, want 2", len(result.Arguments.JVM))
	}
}

func TestMergeVersionsChildJavaVersionOverridesParent(t *testing.T) {
	child := &metadata.VersionDetail{
		ID: "fabric",
		JavaVersion: metadata.JavaVersion{
			Component:    "java-runtime-gamma",
			MajorVersion: 17,
		},
	}
	parent := &metadata.VersionDetail{
		ID: "1.20.1",
		JavaVersion: metadata.JavaVersion{
			Component:    "java-runtime-delta",
			MajorVersion: 21,
		},
	}

	result := metadata.MergeVersions(child, parent)
	if result.JavaVersion.MajorVersion != 17 {
		t.Errorf("JavaVersion.MajorVersion = %d, want 17 (child should override)", result.JavaVersion.MajorVersion)
	}
}

func TestMergeVersionsNilParent(t *testing.T) {
	child := &metadata.VersionDetail{
		ID:          "1.0",
		JavaVersion: metadata.JavaVersion{MajorVersion: 8},
	}

	result := metadata.MergeVersions(child, nil)
	if result.ID != "1.0" {
		t.Errorf("ID = %q, want %q", result.ID, "1.0")
	}
}

func TestMergeVersionsClearsInheritsFrom(t *testing.T) {
	child := &metadata.VersionDetail{
		ID:           "fabric",
		InheritsFrom: "1.20.1",
	}
	parent := &metadata.VersionDetail{
		ID: "1.20.1",
	}

	result := metadata.MergeVersions(child, parent)
	if result.InheritsFrom != "" {
		t.Errorf("InheritsFrom = %q, want empty", result.InheritsFrom)
	}
}

func TestMergeVersionsFillsMainClass(t *testing.T) {
	child := &metadata.VersionDetail{
		ID: "fabric",
	}
	parent := &metadata.VersionDetail{
		ID:        "1.20.1",
		MainClass: "net.minecraft.client.main.Main",
	}

	result := metadata.MergeVersions(child, parent)
	if result.MainClass != "net.minecraft.client.main.Main" {
		t.Errorf("MainClass = %q, want %q", result.MainClass, "net.minecraft.client.main.Main")
	}
}
