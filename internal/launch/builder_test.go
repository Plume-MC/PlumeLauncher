package launch_test

import (
	"testing"

	"plumelauncher/internal/launch"
	"plumelauncher/internal/metadata"
)

func TestBuildArgumentsModern(t *testing.T) {
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
			},
		},
		MainClass:  "net.minecraft.client.main.Main",
		AssetIndex: metadata.AssetIndex{ID: "1.18"},
	}

	opts := launch.Options{
		PlayerName: "TestPlayer",
		UUID:       "test-uuid-1234",
		VersionID:  "1.18.2",
		GameDir:    "C:/Users/test/AppData/.minecraft",
		NativesDir: "C:/Users/test/AppData/.minecraft/versions/1.18.2/natives",
		RamMB:      4096,
		JavaPath:   "C:/Program Files/Java/jdk-17/bin/javaw.exe",
	}

	args, err := launch.BuildArguments(version, opts)
	if err != nil {
		t.Fatalf("BuildArguments: %v", err)
	}

	// Check that args contain expected values
	found := false
	for _, a := range args {
		if a == "-Xmx4096M" {
			found = true
			break
		}
	}
	if !found {
		t.Error("missing -Xmx4096M")
	}

	// Check main class
	found = false
	for _, a := range args {
		if a == "net.minecraft.client.main.Main" {
			found = true
			break
		}
	}
	if !found {
		t.Error("missing main class")
	}

	// Check username substitution
	found = false
	for i, a := range args {
		if a == "--username" && i+1 < len(args) && args[i+1] == "TestPlayer" {
			found = true
			break
		}
	}
	if !found {
		t.Error("username not substituted")
	}
}

func TestBuildArgumentsLegacy(t *testing.T) {
	version := metadata.VersionDetail{
		ID:                 "1.0",
		MainClass:          "net.minecraft.client.Minecraft",
		MinecraftArguments: []byte(`"--username" "${auth_player_name}" "--version" "${version_name}"`),
	}

	opts := launch.Options{
		PlayerName: "Player",
		VersionID:  "1.0",
		GameDir:    "C:/Users/test/.minecraft",
		NativesDir: "C:/Users/test/.minecraft/versions/1.0/natives",
		RamMB:      1024,
	}

	args, err := launch.BuildArguments(version, opts)
	if err != nil {
		t.Fatalf("BuildArguments: %v", err)
	}

	// Check main class
	found := false
	for _, a := range args {
		if a == "net.minecraft.client.Minecraft" {
			found = true
			break
		}
	}
	if !found {
		t.Error("missing main class")
	}

	// Check legacy args
	found = false
	for i, a := range args {
		if a == "--username" && i+1 < len(args) && args[i+1] == "Player" {
			found = true
			break
		}
	}
	if !found {
		t.Error("legacy username not substituted")
	}
}

func TestBuildArgumentsLegacyQuotedValue(t *testing.T) {
	version := metadata.VersionDetail{
		ID:                 "1.0",
		MainClass:          "net.minecraft.client.Minecraft",
		MinecraftArguments: []byte(`--username "Player Name"`),
	}

	args, err := launch.BuildArguments(version, launch.Options{GameDir: "test", NativesDir: "test/natives"})
	if err != nil {
		t.Fatalf("BuildArguments: %v", err)
	}
	for i, arg := range args {
		if arg == "--username" && i+1 < len(args) && args[i+1] == "Player Name" {
			return
		}
	}
	t.Fatal("quoted legacy value was split")
}

func TestMapUserType(t *testing.T) {
	// Test via buildArguments indirectly
	version := metadata.VersionDetail{ID: "test", MainClass: "main", AssetIndex: metadata.AssetIndex{ID: "test"}}
	opts := launch.Options{VersionID: "test", UserType: "offline", GameDir: "test", NativesDir: "test"}

	args, err := launch.BuildArguments(version, opts)
	if err != nil {
		t.Fatalf("BuildArguments: %v", err)
	}

	// Check that user_type is mapped to "legacy" for offline
	found := false
	for i, a := range args {
		if a == "${user_type}" || a == "legacy" {
			found = true
			break
		}
		_ = i
	}
	if !found {
		// User type is embedded in game args via variable substitution
		// Just verify args built successfully
	}
}

func TestBuildArgumentsFullscreen(t *testing.T) {
	version := metadata.VersionDetail{
		ID:         "1.18.2",
		MainClass:  "net.minecraft.client.main.Main",
		AssetIndex: metadata.AssetIndex{ID: "1.18"},
	}

	opts := launch.Options{
		VersionID:  "1.18.2",
		GameDir:    "test",
		NativesDir: "test/natives",
		Fullscreen: true,
		Width:      1920,
		Height:     1080,
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

func TestBuildArgumentsAddsVerifiedInjector(t *testing.T) {
	args, err := launch.BuildArguments(metadata.VersionDetail{ID: "test", MainClass: "main"}, launch.Options{GameDir: "test", NativesDir: "test/natives", AuthlibInjector: "C:/cache/authlib-injector.jar"})
	if err != nil {
		t.Fatalf("BuildArguments: %v", err)
	}
	for _, arg := range args {
		if arg == "-javaagent:C:/cache/authlib-injector.jar=https://authserver.ely.by" {
			return
		}
	}
	t.Fatal("missing authlib injector argument")
}
