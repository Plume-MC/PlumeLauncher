package metadata_test

import (
	"encoding/json"
	"testing"

	"plumelauncher/internal/metadata"
)

func TestArgumentUnmarshalString(t *testing.T) {
	input := `"--username"`
	var arg metadata.Argument
	if err := json.Unmarshal([]byte(input), &arg); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if arg.StringValue != "--username" {
		t.Errorf("StringValue = %q, want %q", arg.StringValue, "--username")
	}
	if arg.Conditional != nil {
		t.Error("Conditional should be nil for string argument")
	}
}

func TestArgumentUnmarshalConditional(t *testing.T) {
	input := `{"rules":[{"action":"allow","os":{"name":"windows"}}],"value":"-Xss1M"}`
	var arg metadata.Argument
	if err := json.Unmarshal([]byte(input), &arg); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if arg.StringValue != "" {
		t.Error("StringValue should be empty for conditional argument")
	}
	if arg.Conditional == nil {
		t.Fatal("Conditional should not be nil")
	}
	if len(arg.Conditional.Rules) != 1 {
		t.Errorf("Rules len = %d, want 1", len(arg.Conditional.Rules))
	}
	if arg.Conditional.Rules[0].Action != "allow" {
		t.Errorf("Rules[0].Action = %q, want %q", arg.Conditional.Rules[0].Action, "allow")
	}
}

func TestArgumentUnmarshalConditionalArrayValue(t *testing.T) {
	input := `{"rules":[{"action":"allow","features":{"has_custom_resolution":true}}],"value":["--width","${resolution_width}","--height","${resolution_height}"]}`
	var arg metadata.Argument
	if err := json.Unmarshal([]byte(input), &arg); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if arg.Conditional == nil {
		t.Fatal("Conditional should not be nil")
	}
	// Value is raw JSON — verify it's an array
	var vals []string
	if err := json.Unmarshal(arg.Conditional.Value, &vals); err != nil {
		t.Fatalf("Unmarshal Value: %v", err)
	}
	if len(vals) != 4 {
		t.Errorf("Value len = %d, want 4", len(vals))
	}
}

func TestVersionDetailUnmarshalModern(t *testing.T) {
	fixture := `{
		"arguments": {
			"game": ["--username", "${auth_player_name}"],
			"jvm": ["-Djava.library.path=${natives_directory}"]
		},
		"assetIndex": {"id": "1.18", "sha1": "abc", "size": 100, "totalSize": 200, "url": "http://example.com/1.18.json"},
		"assets": "1.18",
		"downloads": {
			"client": {"sha1": "def", "size": 300, "url": "http://example.com/client.jar"}
		},
		"id": "1.18.2",
		"javaVersion": {"component": "java-runtime-beta", "majorVersion": 17},
		"libraries": [
			{"name": "com.mojang:logging:1.0.0", "downloads": {"artifact": {"path": "com/mojang/logging/1.0.0/logging-1.0.0.jar", "sha1": "ghi", "size": 400, "url": "http://example.com/logging.jar"}}}
		],
		"mainClass": "net.minecraft.client.main.Main",
		"minimumLauncherVersion": 21,
		"releaseTime": "2022-02-28T10:42:45+00:00",
		"time": "2022-02-28T10:42:45+00:00",
		"type": "release"
	}`

	var detail metadata.VersionDetail
	if err := json.Unmarshal([]byte(fixture), &detail); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if detail.ID != "1.18.2" {
		t.Errorf("ID = %q, want %q", detail.ID, "1.18.2")
	}
	if detail.JavaVersion.MajorVersion != 17 {
		t.Errorf("JavaVersion.MajorVersion = %d, want 17", detail.JavaVersion.MajorVersion)
	}
	if detail.JavaVersion.Component != "java-runtime-beta" {
		t.Errorf("JavaVersion.Component = %q, want %q", detail.JavaVersion.Component, "java-runtime-beta")
	}
	if len(detail.Libraries) != 1 {
		t.Errorf("Libraries len = %d, want 1", len(detail.Libraries))
	}
	if detail.Arguments == nil {
		t.Fatal("Arguments should not be nil")
	}
	if len(detail.Arguments.Game) != 2 {
		t.Errorf("Arguments.Game len = %d, want 2", len(detail.Arguments.Game))
	}
}

func TestVersionDetailUnmarshalLegacy(t *testing.T) {
	fixture := `{
		"assetIndex": {"id": "1.0", "sha1": "abc", "size": 100, "totalSize": 200, "url": "http://example.com/1.0.json"},
		"assets": "1.0",
		"downloads": {
			"client": {"sha1": "def", "size": 300, "url": "http://example.com/client.jar"}
		},
		"id": "1.0",
		"javaVersion": {"component": "java-runtime-alpha", "majorVersion": 8},
		"libraries": [],
		"mainClass": "net.minecraft.client.Minecraft",
		"minimumLauncherVersion": 12,
		"releaseTime": "2009-05-13T15:36:15+00:00",
		"time": "2009-05-13T15:36:15+00:00",
		"type": "release"
	}`

	var detail metadata.VersionDetail
	if err := json.Unmarshal([]byte(fixture), &detail); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if detail.ID != "1.0" {
		t.Errorf("ID = %q, want %q", detail.ID, "1.0")
	}
	if detail.JavaVersion.MajorVersion != 8 {
		t.Errorf("JavaVersion.MajorVersion = %d, want 8", detail.JavaVersion.MajorVersion)
	}
	// Legacy versions have no arguments field
	if detail.Arguments != nil {
		t.Error("Arguments should be nil for legacy version")
	}
}

func TestVersionDetailUnmarshal26Point2(t *testing.T) {
	complianceLevel := 1
	fixture := `{
		"arguments": {
			"game": ["--username", "${auth_player_name}"],
			"jvm": ["-Djava.library.path=${natives_directory}"]
		},
		"assetIndex": {"id": "26.2", "sha1": "abc", "size": 100, "totalSize": 200, "url": "http://example.com/26.2.json"},
		"assets": "26.2",
		"complianceLevel": 1,
		"downloads": {
			"client": {"sha1": "def", "size": 300, "url": "http://example.com/client.jar"}
		},
		"id": "26.2",
		"javaVersion": {"component": "java-runtime-epsilon", "majorVersion": 25},
		"libraries": [],
		"mainClass": "net.minecraft.client.main.Main",
		"minimumLauncherVersion": 21,
		"releaseTime": "2026-06-16T12:03:33+00:00",
		"time": "2026-06-16T12:03:33+00:00",
		"type": "release"
	}`

	var detail metadata.VersionDetail
	if err := json.Unmarshal([]byte(fixture), &detail); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if detail.ID != "26.2" {
		t.Errorf("ID = %q, want %q", detail.ID, "26.2")
	}
	if detail.JavaVersion.MajorVersion != 25 {
		t.Errorf("JavaVersion.MajorVersion = %d, want 25", detail.JavaVersion.MajorVersion)
	}
	if detail.ComplianceLevel == nil || *detail.ComplianceLevel != complianceLevel {
		t.Errorf("ComplianceLevel = %v, want %d", detail.ComplianceLevel, complianceLevel)
	}
}

func TestVersionManifestUnmarshal(t *testing.T) {
	fixture := `{
		"latest": {"release": "26.2", "snapshot": "26.3-snapshot-5"},
		"versions": [
			{"id": "26.2", "type": "release", "url": "http://example.com/26.2.json", "time": "2026-06-16", "releaseTime": "2026-06-16", "sha1": "abc123"},
			{"id": "1.0", "type": "release", "url": "http://example.com/1.0.json", "time": "2009-05-13", "releaseTime": "2009-05-13", "sha1": "def456"}
		]
	}`

	var manifest metadata.VersionManifest
	if err := json.Unmarshal([]byte(fixture), &manifest); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if manifest.Latest.Release != "26.2" {
		t.Errorf("Latest.Release = %q, want %q", manifest.Latest.Release, "26.2")
	}
	if len(manifest.Versions) != 2 {
		t.Errorf("Versions len = %d, want 2", len(manifest.Versions))
	}
	if manifest.Versions[0].ID != "26.2" {
		t.Errorf("Versions[0].ID = %q, want %q", manifest.Versions[0].ID, "26.2")
	}
}

func TestArgumentMarshalRoundtrip(t *testing.T) {
	original := metadata.Argument{StringValue: "--version"}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var roundtripped metadata.Argument
	if err := json.Unmarshal(data, &roundtripped); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if roundtripped.StringValue != "--version" {
		t.Errorf("StringValue = %q, want %q", roundtripped.StringValue, "--version")
	}
}
