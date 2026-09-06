package downloader_test

import (
	"os"
	"path/filepath"
	"testing"

	"plumelauncher/internal/downloader"
	"plumelauncher/internal/metadata"
)

func TestVerifyPlanAllValid(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "lib"), 0o755)
	os.WriteFile(filepath.Join(dir, "lib", "a.jar"), []byte("data"), 0o644)

	plan := &metadata.ArtifactPlan{
		VersionID: "test",
		Artifacts: []metadata.Artifact{
			{Path: "lib/a.jar", Size: 4, Required: true},
		},
	}

	results := downloader.VerifyPlan(dir, plan)
	if len(results) != 1 {
		t.Fatalf("results len = %d, want 1", len(results))
	}
	if !results[0].Valid {
		t.Errorf("expected valid, got missing=%v corrupt=%v err=%v", results[0].Missing, results[0].Corrupt, results[0].Error)
	}
}

func TestVerifyPlanMissing(t *testing.T) {
	dir := t.TempDir()

	plan := &metadata.ArtifactPlan{
		VersionID: "test",
		Artifacts: []metadata.Artifact{
			{Path: "lib/missing.jar", Size: 4, Required: true},
		},
	}

	results := downloader.VerifyPlan(dir, plan)
	if len(results) != 1 {
		t.Fatalf("results len = %d, want 1", len(results))
	}
	if results[0].Valid {
		t.Error("expected invalid for missing file")
	}
	if !results[0].Missing {
		t.Error("expected Missing=true")
	}
}

func TestVerifyPlanSizeMismatch(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "lib"), 0o755)
	os.WriteFile(filepath.Join(dir, "lib", "a.jar"), []byte("data"), 0o644)

	plan := &metadata.ArtifactPlan{
		VersionID: "test",
		Artifacts: []metadata.Artifact{
			{Path: "lib/a.jar", Size: 100, Required: true}, // wrong size
		},
	}

	results := downloader.VerifyPlan(dir, plan)
	if results[0].Valid {
		t.Error("expected invalid for size mismatch")
	}
	if !results[0].Corrupt {
		t.Error("expected Corrupt=true")
	}
}

func TestVerifyArtifactValidWithHash(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "lib"), 0o755)
	os.WriteFile(filepath.Join(dir, "lib", "a.jar"), []byte("hello world"), 0o644)

	artifact := metadata.Artifact{
		Path: "lib/a.jar",
		Size: 11,
		Sha1: "2aae6c35c94fcfb415dbe95f408b9ce91ee846ed",
	}

	status := downloader.VerifyArtifact(filepath.Join(dir, "lib", "a.jar"), artifact)
	if !status.Valid {
		t.Errorf("expected valid, err=%v", status.Error)
	}
}

func TestVerifyArtifactHashMismatch(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "lib"), 0o755)
	os.WriteFile(filepath.Join(dir, "lib", "a.jar"), []byte("wrong data"), 0o644)

	artifact := metadata.Artifact{
		Path: "lib/a.jar",
		Size: 10,
		Sha1: "2aae6c35c94fcfb415dbe95f408b9ce91ee846ed",
	}

	status := downloader.VerifyArtifact(filepath.Join(dir, "lib", "a.jar"), artifact)
	if status.Valid {
		t.Error("expected invalid for hash mismatch")
	}
	if !status.Corrupt {
		t.Error("expected Corrupt=true")
	}
}

func TestVerifyArtifactRejectsMissingIntegrityMetadata(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lib", "a.jar")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	status := downloader.VerifyArtifact(path, metadata.Artifact{Path: "lib/a.jar"})
	if status.Valid || !status.Corrupt {
		t.Fatalf("expected missing integrity metadata to be invalid: %#v", status)
	}
	if status.Error != downloader.ErrMissingIntegrityMetadata {
		t.Fatalf("error = %v, want ErrMissingIntegrityMetadata", status.Error)
	}
}

func TestCheckPlanRequiresHashWhenSizeUnavailable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lib", "a.jar")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("hello world"), 0o644); err != nil {
		t.Fatal(err)
	}
	artifact := metadata.Artifact{Path: "lib/a.jar", Sha1: "2aae6c35c94fcfb415dbe95f408b9ce91ee846ed"}
	plan := &metadata.ArtifactPlan{Artifacts: []metadata.Artifact{artifact}}

	if status := downloader.CheckPlan(dir, plan)[0]; status.Valid {
		t.Fatal("warm check accepted an un-sized artifact without hashing")
	}
	if status := downloader.VerifyPlan(dir, plan)[0]; !status.Valid {
		t.Fatalf("full verification rejected valid hash: %v", status.Error)
	}
}

func TestCheckPlanSkipsHashForWarmLaunch(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "lib"), 0o755)
	os.WriteFile(filepath.Join(dir, "lib", "a.jar"), []byte("wrong data"), 0o644)

	plan := &metadata.ArtifactPlan{
		VersionID: "test",
		Artifacts: []metadata.Artifact{{
			Path: "lib/a.jar",
			Size: 10,
			Sha1: "2aae6c35c94fcfb415dbe95f408b9ce91ee846ed",
		}},
	}

	if status := downloader.CheckPlan(dir, plan)[0]; !status.Valid {
		t.Fatalf("warm launch check marked same-size artifact invalid: %v", status.Error)
	}
	if status := downloader.VerifyPlan(dir, plan)[0]; status.Valid {
		t.Fatal("full verification accepted a hash mismatch")
	}
}

func TestVerifyPlanMixedStatuses(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "lib"), 0o755)
	os.WriteFile(filepath.Join(dir, "lib", "good.jar"), []byte("data"), 0o644)
	// bad.jar exists but wrong size
	os.WriteFile(filepath.Join(dir, "lib", "bad.jar"), []byte("x"), 0o644)

	plan := &metadata.ArtifactPlan{
		VersionID: "test",
		Artifacts: []metadata.Artifact{
			{Path: "lib/good.jar", Size: 4, Required: true},
			{Path: "lib/bad.jar", Size: 100, Required: true},
			{Path: "lib/missing.jar", Size: 4, Required: true},
		},
	}

	results := downloader.VerifyPlan(dir, plan)
	if len(results) != 3 {
		t.Fatalf("results len = %d, want 3", len(results))
	}

	validCount := 0
	for _, r := range results {
		if r.Valid {
			validCount++
		}
	}
	if validCount != 1 {
		t.Errorf("valid count = %d, want 1", validCount)
	}
}
