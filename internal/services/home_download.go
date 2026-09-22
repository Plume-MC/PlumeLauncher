package services

import (
	"context"
	"errors"
	"time"

	"plumelauncher/internal/downloader"
	"plumelauncher/internal/instances"
	"plumelauncher/internal/metadata"
	"plumelauncher/internal/security"
)

func (s *HomeService) InstallInstance(id string) error {
	inst, err := s.Instances.Get(id)
	if err != nil {
		return NewNotFoundError("Instance not found.")
	}
	if inst.State == instances.StatePlanning || inst.State == instances.StateDownloading || inst.State == instances.StateVerifying {
		return NewConflictError("Install is already running for this instance.")
	}

	op, err := s.Registry.Start(id, instances.OpDownload)
	if err != nil {
		return NewConflictError("Install is already running for this instance.")
	}
	if s.Logger != nil {
		s.Logger.Info("install_requested", "instanceId", id, "operationId", op.ID)
	}
	defer func() {
		if current := s.Registry.Get(id); current != nil && current.Status == instances.OpStatusRunning {
			s.Registry.Fail(id)
		}
	}()

	detail, err := s.resolveInstanceDetail(op.CancelContext, inst)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return s.cancelInstall(id, op, inst.State)
		}
		return NewUpstreamError("Unable to load version details. Check your connection and try again.")
	}
	plan := metadata.ResolvePlan(*detail, metadata.CurrentSystem())
	for _, artifact := range plan.Artifacts {
		if err := security.ValidateArtifactURL(artifact.URL); err != nil {
			return NewValidationError("A download address failed validation.", "artifact.url")
		}
	}
	// Pre-verify: skip already-committed artifacts on retry after cancel
	statuses := downloader.VerifyPlan(s.DataRoot, plan)
	var remaining []metadata.Artifact
	for _, status := range statuses {
		if !status.Valid {
			remaining = append(remaining, status.Artifact)
		}
	}
	if err := op.CancelContext.Err(); err != nil {
		return s.cancelInstall(id, op, inst.State)
	}
	if len(remaining) == 0 {
		if err := transitionInstanceState(s.App, s.Instances, id, instances.StateVerifying, op.ID); err != nil {
			return err
		}
		if err := transitionInstanceState(s.App, s.Instances, id, instances.StateReady, op.ID); err != nil {
			return err
		}
		emit(s.App, EventDownloadProgress, DownloadProgressEvent{OperationID: op.ID, InstanceID: id, Status: "completed", FileProgress: len(plan.Artifacts), TotalFiles: len(plan.Artifacts)})
		s.Registry.Complete(id)
		return nil
	}
	plan.Artifacts = remaining
	if err := transitionInstanceState(s.App, s.Instances, id, instances.StatePlanning, op.ID); err != nil {
		return err
	}
	if err := transitionInstanceState(s.App, s.Instances, id, instances.StateDownloading, op.ID); err != nil {
		return err
	}
	if err := op.CancelContext.Err(); err != nil {
		return s.cancelInstall(id, op, inst.State)
	}
	orch := downloader.NewOrchestrator(s.DataRoot, 10)
	var totalBytes int64
	for _, artifact := range plan.Artifacts {
		totalBytes += artifact.Size
	}
	emit(s.App, EventDownloadProgress, DownloadProgressEvent{
		OperationID: op.ID, InstanceID: id, Status: "downloading",
		TotalFiles: len(plan.Artifacts), TotalBytes: totalBytes,
	})
	stopProgress := make(chan struct{})
	go func() {
		ticker := time.NewTicker(250 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				snapshot := orch.Progress()
				emit(s.App, EventDownloadProgress, DownloadProgressEvent{
					OperationID: op.ID, InstanceID: id, Status: "downloading",
					FileProgress: snapshot.CompletedFiles, TotalFiles: snapshot.TotalFiles,
					ByteProgress: snapshot.CompletedBytes, TotalBytes: snapshot.TotalBytes,
					Speed: snapshot.Speed, ETA: snapshot.ETA.Seconds(),
				})
			case <-stopProgress:
				return
			}
		}
	}()
	defer close(stopProgress)
	if err := orch.DownloadPlan(op.CancelContext, plan); err != nil {
		if s.Logger != nil {
			s.Logger.Error("install_failed", "instanceId", id, "operationId", op.ID, "error", err.Error())
		}
		if errors.Is(err, context.Canceled) || s.Registry.Get(id).Status == instances.OpStatusCancelled {
			return s.cancelInstall(id, op, inst.State)
		}
		emit(s.App, EventDownloadProgress, DownloadProgressEvent{OperationID: op.ID, InstanceID: id, Status: "failed", Error: err.Error()})
		_ = transitionInstanceState(s.App, s.Instances, id, instances.StateFailed, op.ID)
		return NewIntegrityError("Unable to download game files. Check your connection and try again.")
	}
	if err := op.CancelContext.Err(); err != nil {
		return s.cancelInstall(id, op, inst.State)
	}
	if err := transitionInstanceState(s.App, s.Instances, id, instances.StateVerifying, op.ID); err != nil {
		return err
	}
	if err := transitionInstanceState(s.App, s.Instances, id, instances.StateReady, op.ID); err != nil {
		return err
	}
	emit(s.App, EventDownloadProgress, DownloadProgressEvent{OperationID: op.ID, InstanceID: id, Status: "completed", FileProgress: len(plan.Artifacts), TotalFiles: len(plan.Artifacts)})
	s.Registry.Complete(id)
	if s.Logger != nil {
		s.Logger.Info("install_completed", "instanceId", id, "operationId", op.ID)
	}
	return nil
}

// RetryInstance retries installation or repair for a failed instance.
func (s *HomeService) RetryInstance(id string) error {
	inst, err := s.Instances.Get(id)
	if err != nil {
		return NewNotFoundError("Instance not found.")
	}
	if inst.State != instances.StateFailed && inst.State != instances.StateCrashed {
		return NewConflictError("Nothing to retry. This instance has no failed install.")
	}
	return s.InstallInstance(id)
}

// VerifyInstance checks the persisted artifact plan for an instance.
func (s *HomeService) VerifyInstance(id string) ([]downloader.VerifyStatus, error) {
	inst, err := s.Instances.Get(id)
	if err != nil {
		return nil, NewNotFoundError("Instance not found.")
	}
	detail, err := s.resolveInstanceDetail(context.Background(), inst)
	if err != nil {
		return nil, NewUpstreamError("Unable to load version details. Check your connection and try again.")
	}
	plan := metadata.ResolvePlan(*detail, metadata.CurrentSystem())
	return downloader.VerifyPlan(s.DataRoot, plan), nil
}

// RepairInstance restores missing or corrupt artifacts for an instance.
func (s *HomeService) RepairInstance(id string) error {
	inst, err := s.Instances.Get(id)
	if err != nil {
		return NewNotFoundError("Instance not found.")
	}
	detail, err := s.resolveInstanceDetail(context.Background(), inst)
	if err != nil {
		return NewUpstreamError("Unable to load version details. Check your connection and try again.")
	}
	plan := metadata.ResolvePlan(*detail, metadata.CurrentSystem())
	return s.repairArtifacts(id, inst, plan)
}

// ensureArtifacts repairs only when an artifact needed to launch is missing or has the wrong size.
func (s *HomeService) ensureArtifacts(id string, inst *instances.Instance, plan *metadata.ArtifactPlan) error {
	for _, status := range downloader.CheckPlan(s.DataRoot, plan) {
		if !status.Valid {
			return s.repairArtifacts(id, inst, plan)
		}
	}
	if inst.State == instances.StateCrashed {
		if err := transitionInstanceState(s.App, s.Instances, id, instances.StateVerifying, ""); err != nil {
			return err
		}
		return transitionInstanceState(s.App, s.Instances, id, instances.StateReady, "")
	}
	return nil
}

func (s *HomeService) repairArtifacts(id string, inst *instances.Instance, plan *metadata.ArtifactPlan) error {
	if s.Registry.IsActive(id) {
		return NewConflictError("Another operation is already running for this instance.")
	}
	op, err := s.Registry.Start(id, instances.OpDownload)
	if err != nil {
		return NewConflictError("Another operation is already running for this instance.")
	}
	defer func() {
		if current := s.Registry.Get(id); current != nil && current.Status == instances.OpStatusRunning {
			s.Registry.Fail(id)
		}
	}()
	for _, artifact := range plan.Artifacts {
		if err := security.ValidateArtifactURL(artifact.URL); err != nil {
			return NewValidationError("A download address failed validation.", "artifact.url")
		}
	}
	if inst.State == instances.StateFailed || inst.State == instances.StateCrashed {
		if err := transitionInstanceState(s.App, s.Instances, id, instances.StatePlanning, op.ID); err != nil {
			return err
		}
		if err := transitionInstanceState(s.App, s.Instances, id, instances.StateDownloading, op.ID); err != nil {
			return err
		}
	}
	emit(s.App, EventDownloadProgress, DownloadProgressEvent{OperationID: op.ID, InstanceID: id, Status: "repairing", TotalFiles: len(plan.Artifacts)})
	if err := downloader.RepairPlan(op.CancelContext, s.DataRoot, plan); err != nil {
		if errors.Is(err, context.Canceled) || s.Registry.Get(id).Status == instances.OpStatusCancelled {
			return s.cancelRepair(id, op, inst.State)
		}
		emit(s.App, EventDownloadProgress, DownloadProgressEvent{OperationID: op.ID, InstanceID: id, Status: "failed", Error: err.Error()})
		if inst.State == instances.StateFailed || inst.State == instances.StateCrashed {
			_ = transitionInstanceState(s.App, s.Instances, id, instances.StateFailed, op.ID)
		}
		return NewIntegrityError("Unable to repair game files. Check your connection and try again.")
	}
	if err := op.CancelContext.Err(); err != nil {
		return s.cancelRepair(id, op, inst.State)
	}
	if inst.State == instances.StateFailed || inst.State == instances.StateCrashed {
		if err := transitionInstanceState(s.App, s.Instances, id, instances.StateVerifying, op.ID); err != nil {
			return err
		}
		if err := transitionInstanceState(s.App, s.Instances, id, instances.StateReady, op.ID); err != nil {
			return err
		}
	}
	s.Registry.Complete(id)
	emit(s.App, EventDownloadProgress, DownloadProgressEvent{OperationID: op.ID, InstanceID: id, Status: "completed", FileProgress: len(plan.Artifacts), TotalFiles: len(plan.Artifacts)})
	return nil
}

func (s *HomeService) cancelInstall(id string, op *instances.Operation, previous instances.InstanceState) error {
	_ = transitionInstanceState(s.App, s.Instances, id, previous, op.ID)
	emit(s.App, EventDownloadProgress, DownloadProgressEvent{OperationID: op.ID, InstanceID: id, Status: "cancelled"})
	return NewCancelledError("Install cancelled.")
}

func (s *HomeService) cancelRepair(id string, op *instances.Operation, previous instances.InstanceState) error {
	if previous == instances.StateFailed || previous == instances.StateCrashed {
		_ = transitionInstanceState(s.App, s.Instances, id, previous, op.ID)
	}
	emit(s.App, EventDownloadProgress, DownloadProgressEvent{OperationID: op.ID, InstanceID: id, Status: "cancelled"})
	return NewCancelledError("Repair cancelled.")
}

// CancelInstance cancels the active download or launch operation.
func (s *HomeService) CancelInstance(id string) error {
	if s.Registry.Get(id) == nil || !s.Registry.IsActive(id) {
		return NewNotFoundError("No active operation for this instance.")
	}
	s.Registry.Cancel(id)
	return nil
}
