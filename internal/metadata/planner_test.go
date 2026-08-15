package metadata_test

import (
	"testing"

	"plumelauncher/internal/metadata"
)

func TestResolvePlan1Point0(t *testing.T) {
	detail := metadata.VersionDetail{
		ID: "1.0",
		AssetIndex: metadata.AssetIndex{
			ID:   "1.0",
			URL:  "http://example.com/1.0.json",
			Size: 100,
			SHA1: "abc",
		},
		Downloads: metadata.Downloads{
			Client: &metadata.DownloadInfo{
				SHA1: "def",
				Size: 300,
				URL:  "http://example.com/client.jar",
				Path: "versions/1.0/client.jar",
			},
		},
		Libraries: []metadata.Library{
			{
				Name: "net.java.dev.jna:jna:3.4.0",
				Downloads: &metadata.LibraryDownloads{
					Artifact: metadata.DownloadInfo{
						SHA1: "ghi",
						Size: 400,
						URL:  "http://example.com/jna.jar",
						Path: "net/java/dev/jna/jna/3.4.0/jna-3.4.0.jar",
					},
				},
			},
		},
		JavaVersion: metadata.JavaVersion{MajorVersion: 8},
	}

	sys := metadata.SystemInfo{OS: "windows", Arch: "x64"}
	plan := metadata.ResolvePlan(detail, sys)

	if plan.VersionID != "1.0" {
		t.Errorf("VersionID = %q, want %q", plan.VersionID, "1.0")
	}

	// Should have: 1 client + 1 library + 1 asset index = 3
	clientCount := 0
	libraryCount := 0
	assetCount := 0
	for _, a := range plan.Artifacts {
		switch a.Role {
		case metadata.RoleClient:
			clientCount++
		case metadata.RoleLibrary:
			libraryCount++
		case metadata.RoleAsset:
			assetCount++
		}
	}
	if clientCount != 1 {
		t.Errorf("client count = %d, want 1", clientCount)
	}
	if libraryCount != 1 {
		t.Errorf("library count = %d, want 1", libraryCount)
	}
	if assetCount != 1 {
		t.Errorf("asset count = %d, want 1", assetCount)
	}
}

func TestResolvePlan1Point12Point2(t *testing.T) {
	detail := metadata.VersionDetail{
		ID: "1.12.2",
		AssetIndex: metadata.AssetIndex{
			ID:   "1.12",
			URL:  "http://example.com/1.12.json",
			Size: 100,
			SHA1: "abc",
		},
		Downloads: metadata.Downloads{
			Client: &metadata.DownloadInfo{
				SHA1: "def",
				Size: 300,
				URL:  "http://example.com/client.jar",
				Path: "versions/1.12.2/client.jar",
			},
		},
		Libraries: []metadata.Library{
			{
				Name: "org.lwjgl:lwjgl:3.2.2",
				Natives: map[string]string{
					"windows": "natives-windows",
					"linux":   "natives-linux",
				},
				Downloads: &metadata.LibraryDownloads{
					Artifact: metadata.DownloadInfo{
						SHA1: "ghi",
						Size: 400,
						URL:  "http://example.com/lwjgl.jar",
						Path: "org/lwjgl/lwjgl/3.2.2/lwjgl-3.2.2.jar",
					},
					Classifiers: map[string]metadata.DownloadInfo{
						"natives-windows": {
							SHA1: "jkl",
							Size: 500,
							URL:  "http://example.com/lwjgl-natives.jar",
							Path: "org/lwjgl/lwjgl/3.2.2/lwjgl-3.2.2-natives-windows.jar",
						},
						"natives-linux": {
							SHA1: "mno",
							Size: 600,
							URL:  "http://example.com/lwjgl-natives-linux.jar",
							Path: "org/lwjgl/lwjgl/3.2.2/lwjgl-3.2.2-natives-linux.jar",
						},
					},
				},
			},
		},
		JavaVersion: metadata.JavaVersion{MajorVersion: 8},
	}

	sys := metadata.SystemInfo{OS: "windows", Arch: "x64"}
	plan := metadata.ResolvePlan(detail, sys)

	// Should have: 1 client + 1 library + 1 native + 1 asset index = 4
	roleCounts := map[metadata.ArtifactRole]int{}
	for _, a := range plan.Artifacts {
		roleCounts[a.Role]++
	}
	if roleCounts[metadata.RoleClient] != 1 {
		t.Errorf("client = %d, want 1", roleCounts[metadata.RoleClient])
	}
	if roleCounts[metadata.RoleLibrary] != 1 {
		t.Errorf("library = %d, want 1", roleCounts[metadata.RoleLibrary])
	}
	if roleCounts[metadata.RoleNative] != 1 {
		t.Errorf("native = %d, want 1", roleCounts[metadata.RoleNative])
	}
	if roleCounts[metadata.RoleAsset] != 1 {
		t.Errorf("asset = %d, want 1", roleCounts[metadata.RoleAsset])
	}
}

func TestResolvePlanRuleFiltering(t *testing.T) {
	detail := metadata.VersionDetail{
		ID:         "1.18.2",
		AssetIndex: metadata.AssetIndex{ID: "1.18", URL: "http://example.com"},
		Downloads: metadata.Downloads{
			Client: &metadata.DownloadInfo{URL: "http://example.com/client.jar", Size: 100, SHA1: "a"},
		},
		Libraries: []metadata.Library{
			{
				Name: "allowed-library",
				Downloads: &metadata.LibraryDownloads{
					Artifact: metadata.DownloadInfo{URL: "http://example.com/allowed.jar", Size: 100, SHA1: "b"},
				},
			},
			{
				Name: "osx-only-library",
				Rules: []metadata.Rule{
					{Action: "allow", OS: &metadata.OSRule{Name: "osx"}},
				},
				Downloads: &metadata.LibraryDownloads{
					Artifact: metadata.DownloadInfo{URL: "http://example.com/osx.jar", Size: 100, SHA1: "c"},
				},
			},
			{
				Name: "windows-only-library",
				Rules: []metadata.Rule{
					{Action: "allow", OS: &metadata.OSRule{Name: "windows"}},
				},
				Downloads: &metadata.LibraryDownloads{
					Artifact: metadata.DownloadInfo{URL: "http://example.com/win.jar", Size: 100, SHA1: "d"},
				},
			},
		},
		JavaVersion: metadata.JavaVersion{MajorVersion: 17},
	}

	sys := metadata.SystemInfo{OS: "windows", Arch: "x64"}
	plan := metadata.ResolvePlan(detail, sys)

	libraryCount := 0
	for _, a := range plan.Artifacts {
		if a.Role == metadata.RoleLibrary {
			libraryCount++
		}
	}
	// Should have: allowed-library + windows-only-library = 2 (osx-only excluded)
	if libraryCount != 2 {
		t.Errorf("library count = %d, want 2", libraryCount)
	}
}

func TestResolvePlanDeduplicatesArtifactPaths(t *testing.T) {
	detail := metadata.VersionDetail{
		ID:         "1.18.2",
		AssetIndex: metadata.AssetIndex{ID: "1.18", URL: "http://example.com/assets"},
		Libraries: []metadata.Library{
			{Name: "org.lwjgl:lwjgl:3.2.2", Downloads: &metadata.LibraryDownloads{Artifact: metadata.DownloadInfo{URL: "http://example.com/lwjgl.jar", Path: "org/lwjgl/lwjgl/3.2.2/lwjgl-3.2.2.jar", SHA1: "same", Size: 10}}},
			{Name: "org.lwjgl:lwjgl:3.2.2", Downloads: &metadata.LibraryDownloads{Artifact: metadata.DownloadInfo{URL: "http://example.com/lwjgl.jar", Path: "org/lwjgl/lwjgl/3.2.2/lwjgl-3.2.2.jar", SHA1: "same", Size: 10}}},
		},
	}

	plan := metadata.ResolvePlan(detail, metadata.SystemInfo{OS: "windows", Arch: "x64"})
	count := 0
	for _, artifact := range plan.Artifacts {
		if artifact.Path == "org/lwjgl/lwjgl/3.2.2/lwjgl-3.2.2.jar" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("LWJGL artifact count = %d, want 1", count)
	}
}

func TestResolvePlanPathContainment(t *testing.T) {
	detail := metadata.VersionDetail{
		ID:         "1.0",
		AssetIndex: metadata.AssetIndex{ID: "1.0", URL: "http://example.com"},
		Downloads: metadata.Downloads{
			Client: &metadata.DownloadInfo{URL: "http://example.com", Size: 100, SHA1: "a", Path: "../../etc/passwd"},
		},
	}

	sys := metadata.SystemInfo{OS: "linux", Arch: "x64"}
	plan := metadata.ResolvePlan(detail, sys)

	for _, a := range plan.Artifacts {
		if len(a.Path) > 0 && (a.Path[0] == '/' || a.Path[:2] == "..") {
			t.Errorf("traversal path detected: %q", a.Path)
		}
	}
}

func TestValidatePathRejectsTraversalComponents(t *testing.T) {
	for _, value := range []string{"../artifact.jar", "nested/../artifact.jar", `nested\\..\\artifact.jar`} {
		if err := metadata.ValidatePath(value); err == nil {
			t.Errorf("ValidatePath(%q) succeeded", value)
		}
	}
}

func TestResolvePlanEmptyLibraries(t *testing.T) {
	detail := metadata.VersionDetail{
		ID:         "1.0",
		AssetIndex: metadata.AssetIndex{ID: "1.0", URL: "http://example.com"},
		Downloads: metadata.Downloads{
			Client: &metadata.DownloadInfo{URL: "http://example.com", Size: 100, SHA1: "a"},
		},
		Libraries: []metadata.Library{},
	}

	sys := metadata.SystemInfo{OS: "windows", Arch: "x64"}
	plan := metadata.ResolvePlan(detail, sys)

	// Should have: 1 client + 1 asset index = 2
	if len(plan.Artifacts) != 2 {
		t.Errorf("artifacts len = %d, want 2", len(plan.Artifacts))
	}
}

func TestPlanSummary(t *testing.T) {
	plan := &metadata.ArtifactPlan{
		VersionID: "1.0",
		Artifacts: []metadata.Artifact{
			{Role: metadata.RoleClient},
			{Role: metadata.RoleLibrary},
			{Role: metadata.RoleLibrary},
			{Role: metadata.RoleAsset},
		},
	}
	summary := metadata.PlanSummary(plan)
	if summary == "" {
		t.Error("PlanSummary returned empty string")
	}
}
