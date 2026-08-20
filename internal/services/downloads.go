package services

import (
	"context"

	"plumelauncher/internal/downloader"
	"plumelauncher/internal/metadata"
	"plumelauncher/internal/security"
)

// DownloadService manages artifact downloads.
type DownloadService struct {
	DataRoot string
}

// DownloadPlan starts downloading all artifacts in the plan.
func (s *DownloadService) DownloadPlan(ctx context.Context, plan *metadata.ArtifactPlan) error {
	for _, artifact := range plan.Artifacts {
		if err := security.ValidateArtifactURL(artifact.URL); err != nil {
			return err
		}
	}
	orch := downloader.NewOrchestrator(s.DataRoot, 10)
	return orch.DownloadPlan(ctx, plan)
}

// VerifyPlan checks all artifacts in the plan.
func (s *DownloadService) VerifyPlan(plan *metadata.ArtifactPlan) []downloader.VerifyStatus {
	return downloader.VerifyPlan(s.DataRoot, plan)
}

// RepairPlan re-downloads missing or corrupt artifacts.
func (s *DownloadService) RepairPlan(ctx context.Context, plan *metadata.ArtifactPlan) error {
	return downloader.RepairPlan(ctx, s.DataRoot, plan)
}
