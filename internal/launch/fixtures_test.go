package launch_test

import (
	"testing"

	"plumelauncher/internal/launch"
	"plumelauncher/internal/metadata"
)

func TestBuildArgumentsModern1Point18Point2(t *testing.T) {
	version := metadata.VersionDetail{
		ID: "1.18.2",
		Arguments: &metadata.Arguments{
			JVM: []metadata.Argument{
				{StringValue: "-Djava.library.path=${natives_directory}"},
			},
			Game: []metadata.Argument{
				{StringValue: "--username"},
				{StringValue: "${auth_player_name}"},
				{StringValue: "--version"},
				{StringValue: "${version_name}"},
				{StringValue: "--gameDir"},
				{StringValue: "${game_directory}"},
				{StringValue: "--assetsDir"},
				{StringValue: "${assets_root}"},
				{StringValue: "--assetIndex"},
				{StringValue: "${assets_index_name}"},
				{StringValue: "--uuid"},
				{StringValue: "${auth_uuid}"},
				{StringValue: "--accessToken"},
				{StringValue: "${auth_access_token}"},
				{StringValue: "--clientId"},
				{StringValue: "${clientid}"},
				{StringValue: "--xuid"},
				{StringValue: "${auth_xuid}"},
				{StringValue: "--userType"},
				{StringValue: "${user_type}"},
				{StringValue: "--versionType"},
				{StringValue: "${version_type}"},
			},
		},
		MainClass:  "net.minecraft.client.main.Main",
		AssetIndex: metadata.AssetIndex{ID: "1.18"},
	}

	opts := launch.Options{
		PlayerName: "TestPlayer",
		UUID:       "test-uuid",
		VersionID:  "1.18.2",
		GameDir:    "C:/test/.minecraft",
		NativesDir: "C:/test/.minecraft/versions/1.18.2/natives",
		AssetsDir:  "C:/test/.minecraft/assets",
		RamMB:      4096,
	}

	args, err := launch.BuildArguments(version, opts)
	if err != nil {
		t.Fatalf("BuildArguments: %v", err)
	}

	// Verify key args
	checkArgs := []string{
		"-Xmx4096M",
		"net.minecraft.client.main.Main",
		"TestPlayer",
		"1.18.2",
		"1.18",
		"legacy",
	}

	for _, expected := range checkArgs {
		found := false
		for _, a := range args {
			if a == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing expected arg: %q", expected)
		}
	}
}

func TestBuildArgumentsLegacy1Point0(t *testing.T) {
	version := metadata.VersionDetail{
		ID:                 "1.0",
		MainClass:          "net.minecraft.client.Minecraft",
		MinecraftArguments: []byte(`"--username" "${auth_player_name}" "--version" "${version_name}" "--gameDir" "${game_directory}"`),
		AssetIndex:         metadata.AssetIndex{ID: "1.0"},
	}

	opts := launch.Options{
		PlayerName: "Steve",
		VersionID:  "1.0",
		GameDir:    "C:/test/.minecraft",
		NativesDir: "C:/test/.minecraft/versions/1.0/natives",
		RamMB:      1024,
	}

	args, err := launch.BuildArguments(version, opts)
	if err != nil {
		t.Fatalf("BuildArguments: %v", err)
	}

	// Verify legacy args
	checkArgs := []string{
		"-Xmx1024M",
		"net.minecraft.client.Minecraft",
		"Steve",
		"1.0",
	}

	for _, expected := range checkArgs {
		found := false
		for _, a := range args {
			if a == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing expected arg: %q", expected)
		}
	}
}

func TestBuildArgumentsResolution(t *testing.T) {
	version := metadata.VersionDetail{
		ID:         "1.21.4",
		MainClass:  "net.minecraft.client.main.Main",
		AssetIndex: metadata.AssetIndex{ID: "19"},
	}

	opts := launch.Options{
		VersionID:  "1.21.4",
		GameDir:    "C:/test/.minecraft",
		NativesDir: "C:/test/.minecraft/versions/1.21.4/natives",
		Width:      1920,
		Height:     1080,
	}

	args, err := launch.BuildArguments(version, opts)
	if err != nil {
		t.Fatalf("BuildArguments: %v", err)
	}

	// Check width/height
	foundWidth := false
	foundHeight := false
	for i, a := range args {
		if a == "1920" && i > 0 && args[i-1] == "--width" {
			foundWidth = true
		}
		if a == "1080" && i > 0 && args[i-1] == "--height" {
			foundHeight = true
		}
	}
	if !foundWidth {
		t.Error("missing --width 1920")
	}
	if !foundHeight {
		t.Error("missing --height 1080")
	}
}

func TestBuildArgumentsFullscreenFixture(t *testing.T) {
	version := metadata.VersionDetail{
		ID:         "1.21.4",
		MainClass:  "net.minecraft.client.main.Main",
		AssetIndex: metadata.AssetIndex{ID: "19"},
	}

	opts := launch.Options{
		VersionID:  "1.21.4",
		GameDir:    "C:/test/.minecraft",
		NativesDir: "C:/test/.minecraft/versions/1.21.4/natives",
		Fullscreen: true,
	}

	args, err := launch.BuildArguments(version, opts)
	if err != nil {
		t.Fatalf("BuildArguments: %v", err)
	}

	found := false
	for _, a := range args {
		if a == "--fullscreen" {
			found = true
			break
		}
	}
	if !found {
		t.Error("missing --fullscreen")
	}
}

func TestBuildArgumentsWrapper(t *testing.T) {
	version := metadata.VersionDetail{
		ID:         "1.21.4",
		MainClass:  "net.minecraft.client.main.Main",
		AssetIndex: metadata.AssetIndex{ID: "19"},
	}

	opts := launch.Options{
		VersionID:  "1.21.4",
		GameDir:    "C:/test/.minecraft",
		NativesDir: "C:/test/.minecraft/versions/1.21.4/natives",
		Wrapper:    []string{"gamemoderun", "--"},
	}

	args, err := launch.BuildArguments(version, opts)
	if err != nil {
		t.Fatalf("BuildArguments: %v", err)
	}

	command, commandArgs, err := launch.BuildCommand("/usr/bin/java", args, opts.Wrapper)
	if err != nil {
		t.Fatalf("BuildCommand: %v", err)
	}
	if command != "gamemoderun" || commandArgs[0] != "--" || commandArgs[1] != "/usr/bin/java" {
		t.Errorf("wrapper command = %q %v", command, commandArgs[:2])
	}
}

func TestBuildArgumentsNoMainClass(t *testing.T) {
	version := metadata.VersionDetail{
		ID:         "1.0",
		AssetIndex: metadata.AssetIndex{ID: "1.0"},
	}

	opts := launch.Options{
		VersionID:  "1.0",
		GameDir:    "C:/test/.minecraft",
		NativesDir: "C:/test/.minecraft/versions/1.0/natives",
	}

	_, err := launch.BuildArguments(version, opts)
	if err == nil {
		t.Fatal("expected error for no main class")
	}
}
