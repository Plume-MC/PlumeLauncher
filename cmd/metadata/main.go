// Command metadata fetches Mojang version metadata and builds an artifact plan.
//
// Usage:
//
//	go run cmd/metadata/main.go <version>
//	go run cmd/metadata/main.go --manifest
//	PLUME_DATA_ROOT=.minecraft-dev go run cmd/metadata/main.go 1.0
//
// Examples:
//
//	go run cmd/metadata/main.go 1.0          # Plan for MC 1.0
//	go run cmd/metadata/main.go 1.18.2       # Plan for MC 1.18.2 (JDK 17)
//	go run cmd/metadata/main.go 26.2          # Plan for MC 26.2 (JDK 25)
//	go run cmd/metadata/main.go --manifest    # Print version manifest
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"plumelauncher/internal/metadata"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <version> | --manifest\n", os.Args[0])
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

	if os.Args[1] == "--manifest" {
		printManifest(client, ctx)
		return
	}

	versionID := os.Args[1]
	printPlan(client, ctx, versionID)
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

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(plan)
}
