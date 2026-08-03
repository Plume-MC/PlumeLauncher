package downloader

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"plumelauncher/internal/metadata"
)

// VerifyStatus is the result of verifying a single artifact.
type VerifyStatus struct {
	Artifact metadata.Artifact
	Valid    bool
	Missing  bool
	Corrupt  bool
	Error    error
}

// VerifyPlan checks all artifacts in a plan against local filesystem.
func VerifyPlan(dataRoot string, plan *metadata.ArtifactPlan) []VerifyStatus {
	results := make([]VerifyStatus, 0, len(plan.Artifacts))

	for _, artifact := range plan.Artifacts {
		fullPath := filepath.Join(dataRoot, artifact.Path)
		status := VerifyArtifact(fullPath, artifact)
		results = append(results, status)
	}

	return results
}

// VerifyArtifact checks a single artifact file.
func VerifyArtifact(fullPath string, artifact metadata.Artifact) VerifyStatus {
	status := VerifyStatus{Artifact: artifact}

	// Check file exists
	info, err := os.Stat(fullPath)
	if err != nil {
		status.Missing = true
		status.Error = err
		return status
	}

	// Check size if known
	if artifact.Size > 0 && info.Size() != artifact.Size {
		status.Corrupt = true
		status.Error = fmt.Errorf("size mismatch: got %d, want %d", info.Size(), artifact.Size)
		return status
	}

	// Check hash if provided
	if artifact.Sha1 != "" {
		ok, err := VerifyFileSHA1(fullPath, artifact.Sha1)
		if err != nil {
			status.Corrupt = true
			status.Error = err
			return status
		}
		if !ok {
			status.Corrupt = true
			status.Error = fmt.Errorf("hash mismatch")
			return status
		}
	}

	status.Valid = true
	return status
}

// RepairPlan re-downloads missing or corrupt artifacts from the plan.
func RepairPlan(ctx context.Context, dataRoot string, plan *metadata.ArtifactPlan) error {
	statuses := VerifyPlan(dataRoot, plan)

	// Collect artifacts that need repair
	var toRepair []metadata.Artifact
	for _, s := range statuses {
		if !s.Valid {
			toRepair = append(toRepair, s.Artifact)
		}
	}

	if len(toRepair) == 0 {
		return nil
	}

	// Build a sub-plan with only broken artifacts
	repairPlan := &metadata.ArtifactPlan{
		VersionID: plan.VersionID + "-repair",
		Artifacts: toRepair,
	}

	orch := NewOrchestrator(dataRoot, 10)
	return orch.DownloadPlan(ctx, repairPlan)
}
