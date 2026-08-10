// Command metadata fetches Mojang version metadata and builds an artifact plan.
//
// Usage:
//
//	go run cmd/metadata/main.go <version>
//	go run cmd/metadata/main.go <version> --download
//	go run cmd/metadata/main.go --manifest
//	go run cmd/metadata/main.go --scan-java
//	go run cmd/metadata/main.go --select-java <mc-version>
//	go run cmd/metadata/main.go --build-args <mc-version>
//	go run cmd/metadata/main.go --launch <mc-version>
//	PLUME_DATA_ROOT=.minecraft-dev go run cmd/metadata/main.go 1.0
//
// Examples:
//
//	go run cmd/metadata/main.go 1.0               # Plan for MC 1.0
//	go run cmd/metadata/main.go 1.21.4 --download  # Download all artifacts
//	go run cmd/metadata/main.go --manifest         # Print version manifest
//	go run cmd/metadata/main.go --scan-java        # Detect installed Java
//	go run cmd/metadata/main.go --select-java 1.21.4  # Select Java for MC version
//	go run cmd/metadata/main.go --build-args 1.21.4    # Show launch command
//	go run cmd/metadata/main.go --launch 1.21.4        # Launch Minecraft
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"plumelauncher/internal/auth"
	javapkg "plumelauncher/internal/java"
	"plumelauncher/internal/downloader"
	"plumelauncher/internal/launch"
	"plumelauncher/internal/metadata"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <version> [--download] | --manifest | --scan-java | --select-java <mc-version>\n", os.Args[0])
		os.Exit(1)
	}

	// Resolve data root
	dataRoot := os.Getenv("PLUME_DATA_ROOT")
	if dataRoot == "" {
		home, _ := os.UserHomeDir()
		dataRoot = filepath.Join(home, ".plume-dev")
	}
	os.MkdirAll(dataRoot, 0o755)

	fmt.Fprintf(os.Stderr, "Data root: %s\n", dataRoot)

	client := metadata.NewClient(dataRoot)
	ctx := context.Background()

	switch os.Args[1] {
	case "--manifest":
		printManifest(client, ctx)
	case "--scan-java":
		scanJava()
	case "--select-java":
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "Usage: %s --select-java <mc-version>\n", os.Args[0])
			os.Exit(1)
		}
		selectJava(os.Args[2])
	case "--build-args":
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "Usage: %s --build-args <mc-version>\n", os.Args[0])
			os.Exit(1)
		}
		buildArgs(client, ctx, dataRoot, os.Args[2])
	case "--launch":
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "Usage: %s --launch <mc-version>\n", os.Args[0])
			os.Exit(1)
		}
		launchGame(client, ctx, dataRoot, os.Args[2])
	default:
		versionID := os.Args[1]
		download := len(os.Args) > 2 && os.Args[2] == "--download"
		if download {
			downloadVersion(client, ctx, dataRoot, versionID)
		} else {
			printPlan(client, ctx, versionID)
		}
	}
}

func scanJava() {
	fmt.Fprintf(os.Stderr, "Scanning Java installations...\n\n")

	installs, err := javapkg.ScanJavaInstallations()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(installs) == 0 {
		fmt.Fprintf(os.Stderr, "No Java installations found.\n")
		return
	}

	fmt.Fprintf(os.Stderr, "Found %d Java installation(s):\n\n", len(installs))
	for i, inst := range installs {
		fmt.Fprintf(os.Stderr, "  [%d] Java %d (%s)\n", i+1, inst.Major, inst.Version)
		fmt.Fprintf(os.Stderr, "      Path: %s\n\n", inst.Path)
	}
}

func selectJava(mcVersion string) {
	fmt.Fprintf(os.Stderr, "Scanning Java installations...\n")

	installs, err := javapkg.ScanJavaInstallations()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(installs) == 0 {
		fmt.Fprintf(os.Stderr, "No Java installations found.\n")
		os.Exit(1)
	}

	required := javapkg.RequiredJavaMajor(mcVersion)
	fmt.Fprintf(os.Stderr, "Minecraft %s requires Java %d\n\n", mcVersion, required)

	result, err := javapkg.SelectJava(installs, mcVersion)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Selected: Java %d (%s)\n", result.Major, result.Version)
	fmt.Fprintf(os.Stderr, "Path: %s\n", result.Path)
}

func printManifest(client *metadata.Client, ctx context.Context) {
	manifest, err := client.FetchManifest(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Latest release: %s\n", manifest.Latest.Release)
	fmt.Fprintf(os.Stderr, "Latest snapshot: %s\n", manifest.Latest.Snapshot)
	fmt.Fprintf(os.Stderr, "Total versions: %d\n\n", len(manifest.Versions))

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(manifest)
}

func printPlan(client *metadata.Client, ctx context.Context, versionID string) {
	_, plan := resolvePlan(client, ctx, versionID)

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(plan)
}

func downloadVersion(client *metadata.Client, ctx context.Context, dataRoot string, versionID string) {
	detail, plan := resolvePlan(client, ctx, versionID)

	fmt.Fprintf(os.Stderr, "Downloading %d artifacts + asset objects...\n\n", len(plan.Artifacts))

	orch := downloader.NewOrchestrator(dataRoot, 10)
	start := time.Now()

	// Run download in a goroutine to print progress
	done := make(chan error, 1)
	go func() {
		if err := orch.DownloadPlan(ctx, plan); err != nil {
			done <- err
			return
		}
		if err := orch.DownloadAssets(ctx, *detail); err != nil {
			done <- err
			return
		}
		done <- orch.WaitForDownloads()
	}()

	// Print progress every 500ms
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case err := <-done:
			elapsed := time.Since(start)
			snap := orch.Progress()
			if err != nil {
				fmt.Fprintf(os.Stderr, "\nError: %v\n", err)
				fmt.Fprintf(os.Stderr, "Downloaded %d/%d files (%.1f MB) in %s\n",
					snap.CompletedFiles, snap.TotalFiles,
					float64(snap.CompletedBytes)/1024/1024, elapsed.Round(time.Second))
				os.Exit(1)
			}
			fmt.Fprintf(os.Stderr, "\nDone! Downloaded %d files (%.1f MB) in %s\n",
				snap.CompletedFiles,
				float64(snap.CompletedBytes)/1024/1024,
				elapsed.Round(time.Second))
			return
		case <-ticker.C:
			snap := orch.Progress()
			speed := snap.Speed / 1024 / 1024
			fmt.Fprintf(os.Stderr, "\r  %d/%d files | %.1f/%.1f MB | %.1f MB/s | ETA %s",
				snap.CompletedFiles, snap.TotalFiles,
				float64(snap.CompletedBytes)/1024/1024,
				float64(snap.TotalBytes)/1024/1024,
				speed,
				snap.ETA.Round(time.Second))
		}
	}
}

func resolvePlan(client *metadata.Client, ctx context.Context, versionID string) (*metadata.VersionDetail, *metadata.ArtifactPlan) {
	fmt.Fprintf(os.Stderr, "Resolving version %s...\n", versionID)

	detail, err := client.ResolveVersionChain(ctx, versionID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Version: %s (Java %d, %s)\n", detail.ID, detail.JavaVersion.MajorVersion, detail.JavaVersion.Component)
	fmt.Fprintf(os.Stderr, "Libraries: %d\n", len(detail.Libraries))

	sys := metadata.CurrentSystem()
	fmt.Fprintf(os.Stderr, "System: %s/%s\n\n", sys.OS, sys.Arch)

	plan := metadata.ResolvePlan(*detail, sys)

	fmt.Fprintf(os.Stderr, "%s\n\n", metadata.PlanSummary(plan))

	return detail, plan
}

func buildArgs(client *metadata.Client, ctx context.Context, dataRoot string, versionID string) {
	detail, _ := resolvePlan(client, ctx, versionID)

	absDataRoot, _ := filepath.Abs(dataRoot)
	gameDir := absDataRoot
	nativesDir := filepath.Join(gameDir, "versions", detail.ID, "natives")

	opts := launch.Options{
		PlayerName: "Player",
		UUID:       auth.OfflineUUID("Player"),
		AccessToken: "0",
		UserType:   "offline",
		VersionID:  detail.ID,
		GameDir:    gameDir,
		AssetsDir:  filepath.Join(gameDir, "assets"),
		NativesDir: nativesDir,
		RamMB:      4096,
		Width:      854,
		Height:     480,
	}

	javaPath := ""
	installs, _ := javapkg.ScanJavaInstallations()
	if len(installs) > 0 {
		javaPath = installs[0].Path
	}

	args, err := launch.BuildArguments(*detail, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Java path: %s\n\n", javaPath)
	fmt.Fprintf(os.Stderr, "Launch command:\n")
	fmt.Fprintf(os.Stderr, "  \"%s\" \\\n", javaPath)
	for i, arg := range args {
		if i < len(args)-1 {
			fmt.Fprintf(os.Stderr, "    %s \\\n", arg)
		} else {
			fmt.Fprintf(os.Stderr, "    %s\n", arg)
		}
	}
}

func launchGame(client *metadata.Client, ctx context.Context, dataRoot string, versionID string) {
	detail, plan := resolvePlan(client, ctx, versionID)

	// Verify artifacts exist
	statuses := downloader.VerifyPlan(dataRoot, plan)
	validCount := 0
	for _, s := range statuses {
		if s.Valid {
			validCount++
		}
	}
	if validCount < len(statuses) {
		fmt.Fprintf(os.Stderr, "Warning: %d/%d artifacts missing or corrupt. Run --download first.\n",
			len(statuses)-validCount, len(statuses))
	}

	// Resolve to absolute paths for Java process
	absDataRoot, _ := filepath.Abs(dataRoot)
	gameDir := absDataRoot
	nativesDir := filepath.Join(gameDir, "versions", detail.ID, "natives")

	// Extract natives from downloaded JARs
	fmt.Fprintf(os.Stderr, "Extracting natives...\n")
	os.RemoveAll(nativesDir)
	os.MkdirAll(nativesDir, 0o755)
	sys := metadata.CurrentSystem()
	extracted := 0
	for _, lib := range detail.Libraries {
		if !metadata.ShouldDownload(lib.Rules, sys) {
			continue
		}
		nativePath, ok := metadata.ResolveNativePath(lib, sys)
		if !ok {
			continue
		}
		jarPath := filepath.Join(gameDir, nativePath)
		if err := launch.ExtractNatives(jarPath, nativesDir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to extract %s: %v\n", lib.Name, err)
			continue
		}
		extracted++
	}
	fmt.Fprintf(os.Stderr, "Extracted %d native JARs to %s\n\n", extracted, nativesDir)

	opts := launch.Options{
		PlayerName: "Player",
		UUID:       auth.OfflineUUID("Player"),
		AccessToken: "0",
		UserType:   "offline",
		VersionID:  detail.ID,
		GameDir:    gameDir,
		AssetsDir:  filepath.Join(gameDir, "assets"),
		NativesDir: nativesDir,
		RamMB:      4096,
		Width:      854,
		Height:     480,
	}

	javaPath := ""
	installs, _ := javapkg.ScanJavaInstallations()
	if len(installs) > 0 {
		javaPath = installs[0].Path
	}
	if javaPath == "" {
		fmt.Fprintf(os.Stderr, "Error: no Java installation found\n")
		os.Exit(1)
	}

	args, err := launch.BuildArguments(*detail, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Launching Minecraft %s with Java %s...\n", versionID, javaPath)
	fmt.Fprintf(os.Stderr, "Press Ctrl+C to stop.\n\n")

	cmd, err := launch.Launch(javaPath, args, gameDir, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error launching: %v\n", err)
		os.Exit(1)
	}

	// Handle Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Fprintf(os.Stderr, "\nStopping game...\n")
		launch.Stop(cmd)
	}()

	// Monitor stdout/stderr
	err = launch.Monitor(cmd, func(line string, isStderr bool) {
		if isStderr {
			fmt.Fprintf(os.Stderr, "[ERROR] %s\n", line)
		} else {
			fmt.Fprintf(os.Stderr, "[GAME] %s\n", line)
		}
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "\nGame exited: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "\nGame exited successfully.\n")
}
