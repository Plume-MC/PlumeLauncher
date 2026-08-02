package metadata_test

import (
	"encoding/json"
	"testing"

	"plumelauncher/internal/metadata"
)

// Minimal fixture for each era bucket (not full Mojang JSON — just the fields the planner needs)

var fixture1_0 = `{
	"assetIndex": {"id": "1.0", "sha1": "8576278649375418413", "size": 468892705, "url": "https://piston-meta.mojang.com/v1/packages/8576278649375418413/1.0.json"},
	"assets": "1.0",
	"downloads": {"client": {"sha1": "0b9165852107524e06a0550be05c57e83b74430f", "size": 522395, "url": "https://s3.amazonaws.com/Minecraft.Download/versions/1.0/minecraft.jar", "path": "versions/1.0/minecraft.jar"}},
	"id": "1.0",
	"javaVersion": {"component": "java-runtime-alpha", "majorVersion": 8},
	"libraries": [
		{"name": "net.java.dev.jna:jna:3.4.0", "downloads": {"artifact": {"sha1": "8916b052b5621b9b40529e4a13c781ac7b46e178", "size": 348948, "url": "https://libraries.minecraft.net/net/java/dev/jna/jna/3.4.0/jna-3.4.0.jar", "path": "net/java/dev/jna/jna/3.4.0/jna-3.4.0.jar"}}},
		{"name": "net.java.dev.jna:jna-platform:3.4.0", "downloads": {"artifact": {"sha1": "6c78dd79c08e4d1a15a47a26d1c62e00c88de355", "size": 575500, "url": "https://libraries.minecraft.net/net/java/dev/jna/jna-platform/3.4.0/jna-platform-3.4.0.jar", "path": "net/java/dev/jna/jna-platform/3.4.0/jna-platform-3.4.0.jar"}}},
		{"name": "com.mojang:authlib:1.3", "downloads": {"artifact": {"sha1": "0b4e815d3e450ae34c039c373de21c01e38621b0", "size": 64560, "url": "https://libraries.minecraft.net/com/mojang/authlib/1.3/authlib-1.3.jar", "path": "com/mojang/authlib/1.3/authlib-1.3.jar"}}}
	],
	"mainClass": "net.minecraft.client.Minecraft",
	"minimumLauncherVersion": 12,
	"releaseTime": "2009-05-13T15:36:15+00:00",
	"time": "2009-05-13T15:36:15+00:00",
	"type": "release"
}`

var fixture1_12_2 = `{
	"arguments": {
		"game": ["--username", "${auth_player_name}", "--version", "${version_name}"],
		"jvm": ["-Djava.library.path=${natives_directory}"]
	},
	"assetIndex": {"id": "1.12", "sha1": "bdf48ef6b5d0d23bbb02e17d04865216179f510a", "size": 383160, "totalSize": 194796828, "url": "https://piston-meta.mojang.com/v1/packages/bdf48ef6b5d0d23bbb02e17d04865216179f510a/1.12.json"},
	"assets": "1.12",
	"downloads": {"client": {"sha1": "822ad727d20e5c630f150c5d1471b00136350e42", "size": 8895783, "url": "https://piston-data.mojang.com/v1/objects/822ad727d20e5c630f150c5d1471b00136350e42/client.jar", "path": "versions/1.12.2/1.12.2.jar"}},
	"id": "1.12.2",
	"javaVersion": {"component": "java-runtime-alpha", "majorVersion": 8},
	"libraries": [
		{"name": "org.lwjgl:lwjgl:3.2.2", "natives": {"linux": "natives-linux", "windows": "natives-windows"}, "downloads": {"artifact": {"sha1": "8ad6294407e15780b43e84929c40e4c5e997972e", "size": 321900, "url": "https://libraries.minecraft.net/org/lwjgl/lwjgl/3.2.2/lwjgl-3.2.2.jar", "path": "org/lwjgl/lwjgl/3.2.2/lwjgl-3.2.2.jar"}, "classifiers": {"natives-linux": {"sha1": "ae7976827ca2a3741f6b9a843a89bacd637af350", "size": 124776, "url": "https://libraries.minecraft.net/org/lwjgl/lwjgl/3.2.2/lwjgl-3.2.2-natives-linux.jar", "path": "org/lwjgl/lwjgl/3.2.2/lwjgl-3.2.2-natives-linux.jar"}, "natives-windows": {"sha1": "05359f3aa50d36352815fc662ea73e1c00d22170", "size": 279593, "url": "https://libraries.minecraft.net/org/lwjgl/lwjgl/3.2.2/lwjgl-3.2.2-natives-windows.jar", "path": "org/lwjgl/lwjgl/3.2.2/lwjgl-3.2.2-natives-windows.jar"}}}, "rules": [{"action": "allow"}, {"action": "disallow", "os": {"name": "osx"}}]},
		{"name": "com.mojang:text2speech:1.12.4", "natives": {"linux": "natives-linux", "windows": "natives-windows"}, "extract": {"exclude": ["META-INF/"]}, "downloads": {"artifact": {"sha1": "1f618f522dbdd93218c270bcfd8f8dd84be31717", "size": 12874, "url": "https://libraries.minecraft.net/com/mojang/text2speech/1.12.4/text2speech-1.12.4.jar", "path": "com/mojang/text2speech/1.12.4/text2speech-1.12.4.jar"}, "classifiers": {"natives-linux": {"sha1": "9571b1360a268311d7fa625614186965914f0215", "size": 7833, "url": "https://libraries.minecraft.net/com/mojang/text2speech/1.12.4/text2speech-1.12.4-natives-linux.jar", "path": "com/mojang/text2speech/1.12.4/text2speech-1.12.4-natives-linux.jar"}, "natives-windows": {"sha1": "7e37c535186a058d730ec03491182fae2efb57be", "size": 81379, "url": "https://libraries.minecraft.net/com/mojang/text2speech/1.12.4/text2speech-1.12.4-natives-windows.jar", "path": "com/mojang/text2speech/1.12.4/text2speech-1.12.4-natives-windows.jar"}}}}
	],
	"mainClass": "net.minecraft.client.main.Main",
	"minimumLauncherVersion": 21,
	"releaseTime": "2017-09-18T08:36:14+00:00",
	"time": "2017-09-18T08:36:14+00:00",
	"type": "release"
}`

var fixture1_18_2 = `{
	"arguments": {"game": ["--username", "${auth_player_name}"], "jvm": ["-Djava.library.path=${natives_directory}"]},
	"assetIndex": {"id": "1.18", "sha1": "d31a2e85ae149dd1b1a7070b22cb8887892fda6c", "size": 348724, "totalSize": 468892705, "url": "https://piston-meta.mojang.com/v1/packages/d31a2e85ae149dd1b1a7070b22cb8887892fda6c/1.18.json"},
	"assets": "1.18",
	"downloads": {"client": {"sha1": "2e9a3e3107cca00d6bc9c97bf7d149cae163ef21", "size": 20259661, "url": "https://piston-data.mojang.com/v1/objects/2e9a3e3107cca00d6bc9c97bf7d149cae163ef21/client.jar", "path": "versions/1.18.2/1.18.2.jar"}},
	"id": "1.18.2",
	"javaVersion": {"component": "java-runtime-beta", "majorVersion": 17},
	"libraries": [
		{"name": "org.lwjgl:lwjgl:3.2.2", "natives": {"linux": "natives-linux", "windows": "natives-windows"}, "downloads": {"artifact": {"sha1": "8ad6294407e15780b43e84929c40e4c5e997972e", "size": 321900, "url": "https://libraries.minecraft.net/org/lwjgl/lwjgl/3.2.2/lwjgl-3.2.2.jar", "path": "org/lwjgl/lwjgl/3.2.2/lwjgl-3.2.2.jar"}, "classifiers": {"natives-linux": {"sha1": "ae7976827ca2a3741f6b9a843a89bacd637af350", "size": 124776, "url": "https://libraries.minecraft.net/org/lwjgl/lwjgl/3.2.2/lwjgl-3.2.2-natives-linux.jar", "path": "org/lwjgl/lwjgl/3.2.2/lwjgl-3.2.2-natives-linux.jar"}, "natives-windows": {"sha1": "05359f3aa50d36352815fc662ea73e1c00d22170", "size": 279593, "url": "https://libraries.minecraft.net/org/lwjgl/lwjgl/3.2.2/lwjgl-3.2.2-natives-windows.jar", "path": "org/lwjgl/lwjgl/3.2.2/lwjgl-3.2.2-natives-windows.jar"}}}, "rules": [{"action": "allow"}, {"action": "disallow", "os": {"name": "osx"}}]},
		{"name": "com.mojang:authlib:3.3.39", "downloads": {"artifact": {"sha1": "289405e70c0917eaeac017f7fba9adb4427baa36", "size": 98740, "url": "https://libraries.minecraft.net/com/mojang/authlib/3.3.39/authlib-3.3.39.jar", "path": "com/mojang/authlib/3.3.39/authlib-3.3.39.jar"}}}
	],
	"mainClass": "net.minecraft.client.main.Main",
	"minimumLauncherVersion": 21,
	"releaseTime": "2022-02-28T10:42:45+00:00",
	"time": "2022-02-28T10:42:45+00:00",
	"type": "release"
}`

var fixture1_21_4 = `{
	"arguments": {"game": ["--username", "${auth_player_name}"], "jvm": ["-Djava.library.path=${natives_directory}"]},
	"assetIndex": {"id": "1.21.4", "sha1": "abc123", "size": 500000, "totalSize": 600000000, "url": "https://piston-meta.mojang.com/v1/packages/abc123/1.21.4.json"},
	"assets": "1.21.4",
	"downloads": {"client": {"sha1": "def456", "size": 25000000, "url": "https://piston-data.mojang.com/v1/objects/def456/client.jar", "path": "versions/1.21.4/1.21.4.jar"}},
	"id": "1.21.4",
	"javaVersion": {"component": "java-runtime-delta", "majorVersion": 21},
	"libraries": [],
	"mainClass": "net.minecraft.client.main.Main",
	"minimumLauncherVersion": 21,
	"releaseTime": "2024-12-03T00:00:00+00:00",
	"time": "2024-12-03T00:00:00+00:00",
	"type": "release"
}`

var fixture26_2 = `{
	"arguments": {"game": ["--username", "${auth_player_name}"], "jvm": ["-Djava.library.path=${natives_directory}"]},
	"assetIndex": {"id": "26.2", "sha1": "abc123", "size": 500000, "totalSize": 600000000, "url": "https://piston-meta.mojang.com/v1/packages/abc123/26.2.json"},
	"assets": "26.2",
	"complianceLevel": 1,
	"downloads": {"client": {"sha1": "def456", "size": 30000000, "url": "https://piston-data.mojang.com/v1/objects/def456/client.jar", "path": "versions/26.2/26.2.jar"}},
	"id": "26.2",
	"javaVersion": {"component": "java-runtime-epsilon", "majorVersion": 25},
	"libraries": [],
	"mainClass": "net.minecraft.client.main.Main",
	"minimumLauncherVersion": 21,
	"releaseTime": "2026-06-16T12:03:33+00:00",
	"time": "2026-06-16T12:03:33+00:00",
	"type": "release"
}`

func parseDetail(t *testing.T, raw string) metadata.VersionDetail {
	t.Helper()
	var d metadata.VersionDetail
	if err := json.Unmarshal([]byte(raw), &d); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
	return d
}

func TestPlan1Point0(t *testing.T) {
	detail := parseDetail(t, fixture1_0)
	sys := metadata.SystemInfo{OS: "windows", Arch: "x64"}
	plan := metadata.ResolvePlan(detail, sys)

	// 1.0: client + 3 libraries + 1 asset index = 5
	if len(plan.Artifacts) != 5 {
		t.Fatalf("artifacts len = %d, want 5 (client=1, libs=3, assets=1)", len(plan.Artifacts))
	}
	if plan.VersionID != "1.0" {
		t.Errorf("VersionID = %q", plan.VersionID)
	}
	// No natives in 1.0
	for _, a := range plan.Artifacts {
		if a.Role == metadata.RoleNative {
			t.Error("1.0 should have no natives")
		}
	}
}

func TestPlan1Point12Point2(t *testing.T) {
	detail := parseDetail(t, fixture1_12_2)
	sys := metadata.SystemInfo{OS: "windows", Arch: "x64"}
	plan := metadata.ResolvePlan(detail, sys)

	// 1.12.2: client + 2 libraries + 2 natives (lwjgl + text2speech) + 1 asset index = 6
	roleCounts := map[metadata.ArtifactRole]int{}
	for _, a := range plan.Artifacts {
		roleCounts[a.Role]++
	}
	if roleCounts[metadata.RoleClient] != 1 {
		t.Errorf("client = %d, want 1", roleCounts[metadata.RoleClient])
	}
	if roleCounts[metadata.RoleLibrary] != 2 {
		t.Errorf("library = %d, want 2", roleCounts[metadata.RoleLibrary])
	}
	if roleCounts[metadata.RoleNative] != 2 {
		t.Errorf("native = %d, want 2", roleCounts[metadata.RoleNative])
	}
	if roleCounts[metadata.RoleAsset] != 1 {
		t.Errorf("asset = %d, want 1", roleCounts[metadata.RoleAsset])
	}
}

func TestPlan1Point18Point2(t *testing.T) {
	detail := parseDetail(t, fixture1_18_2)
	sys := metadata.SystemInfo{OS: "windows", Arch: "x64"}
	plan := metadata.ResolvePlan(detail, sys)

	// 1.18.2: client + 2 libraries + 1 native (lwjgl only) + 1 asset index = 5
	roleCounts := map[metadata.ArtifactRole]int{}
	for _, a := range plan.Artifacts {
		roleCounts[a.Role]++
	}
	if roleCounts[metadata.RoleClient] != 1 {
		t.Errorf("client = %d, want 1", roleCounts[metadata.RoleClient])
	}
	if roleCounts[metadata.RoleNative] != 1 {
		t.Errorf("native = %d, want 1", roleCounts[metadata.RoleNative])
	}
}

func TestPlan1Point21Point4(t *testing.T) {
	detail := parseDetail(t, fixture1_21_4)
	sys := metadata.SystemInfo{OS: "windows", Arch: "x64"}
	plan := metadata.ResolvePlan(detail, sys)

	// 1.21.4: client + 0 libraries + 1 asset index = 2
	if len(plan.Artifacts) != 2 {
		t.Errorf("artifacts len = %d, want 2", len(plan.Artifacts))
	}
	if detail.JavaVersion.MajorVersion != 21 {
		t.Errorf("JavaVersion = %d, want 21", detail.JavaVersion.MajorVersion)
	}
}

func TestPlan26Point2(t *testing.T) {
	detail := parseDetail(t, fixture26_2)
	sys := metadata.SystemInfo{OS: "windows", Arch: "x64"}
	plan := metadata.ResolvePlan(detail, sys)

	// 26.2: client + 0 libraries + 1 asset index = 2
	if len(plan.Artifacts) != 2 {
		t.Errorf("artifacts len = %d, want 2", len(plan.Artifacts))
	}
	if detail.JavaVersion.MajorVersion != 25 {
		t.Errorf("JavaVersion = %d, want 25", detail.JavaVersion.MajorVersion)
	}
	if detail.ComplianceLevel == nil || *detail.ComplianceLevel != 1 {
		t.Errorf("ComplianceLevel = %v, want 1", detail.ComplianceLevel)
	}
}

func TestPlanRuleFilteringWindows(t *testing.T) {
	detail := parseDetail(t, fixture1_12_2)
	sys := metadata.SystemInfo{OS: "windows", Arch: "x64"}
	plan := metadata.ResolvePlan(detail, sys)

	// lwjgl has rule: allow all, disallow osx → should be included on windows
	// text2speech has no rule → included
	libraryCount := 0
	for _, a := range plan.Artifacts {
		if a.Role == metadata.RoleLibrary {
			libraryCount++
		}
	}
	if libraryCount != 2 {
		t.Errorf("library count on windows = %d, want 2", libraryCount)
	}
}

func TestPlanRuleFilteringOsx(t *testing.T) {
	detail := parseDetail(t, fixture1_12_2)
	sys := metadata.SystemInfo{OS: "osx", Arch: "x64"}
	plan := metadata.ResolvePlan(detail, sys)

	// lwjgl has rule: disallow osx → should be excluded on osx
	// text2speech has no rule → included
	libraryCount := 0
	for _, a := range plan.Artifacts {
		if a.Role == metadata.RoleLibrary {
			libraryCount++
		}
	}
	if libraryCount != 1 {
		t.Errorf("library count on osx = %d, want 1 (text2speech only)", libraryCount)
	}
}

func TestPlanNoTraversalPaths(t *testing.T) {
	fixtures := []string{fixture1_0, fixture1_12_2, fixture1_18_2, fixture1_21_4, fixture26_2}
	sys := metadata.SystemInfo{OS: "windows", Arch: "x64"}

	for _, raw := range fixtures {
		detail := parseDetail(t, raw)
		plan := metadata.ResolvePlan(detail, sys)
		for _, a := range plan.Artifacts {
			if len(a.Path) > 0 && (a.Path[0] == '/' || len(a.Path) >= 2 && a.Path[:2] == "..") {
				t.Errorf("traversal path in %s: %q", detail.ID, a.Path)
			}
		}
	}
}
