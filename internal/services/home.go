package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"plumelauncher/internal/auth"
	"plumelauncher/internal/downloader"
	"plumelauncher/internal/instances"
	"plumelauncher/internal/java"
	"plumelauncher/internal/launch"
	"plumelauncher/internal/metadata"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// HomeService exposes instance actions in terms of persisted instance IDs.
// The frontend never constructs artifact plans or launch command arguments.
type HomeService struct {
	DataRoot  string
	Defaults  instances.LauncherDefaults
	Instances *instances.Manager
	Registry  *instances.Registry
	Launch    *LaunchService
	Accounts  *AccountService
	App       *application.App
}

var selectableVersions = []string{"1.21.4", "1.20.4", "1.18.2", "1.16.5", "1.12.2", "1.8.9", "1.7.10"}

func (s *HomeService) ListInstances() ([]instances.Instance, error) {
	return s.Instances.List()
}

// SupportedVersions returns only versions listed by the selected loader's metadata.
func (s *HomeService) SupportedVersions(loader string) ([]string, error) {
	if loader == "" || loader == "vanilla" {
		return selectableVersions, nil
	}
	if loader != "fabric" && loader != "quilt" {
		return nil, NewValidationError("invalid loader", "loader")
	}
	client := metadata.NewClient(s.DataRoot)
	versions := make([]string, 0, len(selectableVersions))
	for _, version := range selectableVersions {
		supported, err := client.SupportsLoaderVersion(context.Background(), loader, version)
		if err != nil {
			return nil, NewUpstreamError(fmt.Sprintf("resolve %s metadata: %v", loader, err))
		}
		if supported {
			versions = append(versions, version)
		}
	}
	return versions, nil
}

func (s *HomeService) CreateInstance(name, version, loader string) (*instances.Instance, error) {
	loaderType, err := parseLoader(loader)
	if err != nil {
		return nil, err
	}
	return s.Instances.Create(name, version, loaderType)
}

func (s *HomeService) DeleteInstance(id string) error {
	return s.Instances.Delete(id)
}

func (s *HomeService) InstallInstance(id string) error {
	inst, err := s.Instances.Get(id)
	if err != nil {
		return NewNotFoundError("instance not found")
	}
	if inst.State == instances.StatePlanning || inst.State == instances.StateDownloading || inst.State == instances.StateVerifying {
		return NewConflictError("instance install is already active")
	}

	op, err := s.Registry.Start(id, instances.OpDownload)
	if err != nil {
		return NewConflictError(err.Error())
	}
	defer func() {
		if current := s.Registry.Get(id); current != nil && current.Status == instances.OpStatusRunning {
			s.Registry.Fail(id)
		}
	}()

	detail, err := s.resolveInstanceDetail(context.Background(), inst)
	if err != nil {
		return NewUpstreamError(fmt.Sprintf("resolve metadata: %v", err))
	}
	plan := metadata.ResolvePlan(*detail, metadata.CurrentSystem())
	if err := s.Instances.UpdateState(id, instances.StatePlanning); err != nil {
		return err
	}
	if err := s.Instances.UpdateState(id, instances.StateDownloading); err != nil {
		return err
	}
	orch := downloader.NewOrchestrator(s.DataRoot, 10)
	var totalBytes int64
	for _, artifact := range plan.Artifacts {
		totalBytes += artifact.Size
	}
	emit(s.App, EventDownloadProgress, DownloadProgressEvent{
		OperationID: id, InstanceID: id, Status: "downloading",
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
					OperationID: id, InstanceID: id, Status: "downloading",
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
		emit(s.App, EventDownloadProgress, DownloadProgressEvent{OperationID: id, InstanceID: id, Status: "failed", Error: err.Error()})
		_ = s.Instances.UpdateState(id, instances.StateFailed)
		return err
	}
	if err := orch.DownloadAssets(op.CancelContext, *detail); err != nil {
		_ = s.Instances.UpdateState(id, instances.StateFailed)
		return err
	}
	if err := s.Instances.UpdateState(id, instances.StateVerifying); err != nil {
		return err
	}
	if err := s.Instances.UpdateState(id, instances.StateReady); err != nil {
		return err
	}
	emit(s.App, EventDownloadProgress, DownloadProgressEvent{OperationID: id, InstanceID: id, Status: "completed", FileProgress: len(plan.Artifacts), TotalFiles: len(plan.Artifacts)})
	s.Registry.Complete(id)
	return nil
}

// VerifyInstance checks the persisted artifact plan for an instance.
func (s *HomeService) VerifyInstance(id string) ([]downloader.VerifyStatus, error) {
	inst, err := s.Instances.Get(id)
	if err != nil {
		return nil, NewNotFoundError("instance not found")
	}
	detail, err := s.resolveInstanceDetail(context.Background(), inst)
	if err != nil {
		return nil, NewUpstreamError(fmt.Sprintf("resolve metadata: %v", err))
	}
	plan := metadata.ResolvePlan(*detail, metadata.CurrentSystem())
	return downloader.VerifyPlan(s.DataRoot, plan), nil
}

// RepairInstance restores missing or corrupt artifacts for an instance.
func (s *HomeService) RepairInstance(id string) error {
	inst, err := s.Instances.Get(id)
	if err != nil {
		return NewNotFoundError("instance not found")
	}
	if s.Registry.IsActive(id) {
		return NewConflictError("instance already has an active operation")
	}
	detail, err := s.resolveInstanceDetail(context.Background(), inst)
	if err != nil {
		return NewUpstreamError(fmt.Sprintf("resolve metadata: %v", err))
	}
	plan := metadata.ResolvePlan(*detail, metadata.CurrentSystem())
	return downloader.RepairPlan(context.Background(), s.DataRoot, plan)
}

// CancelInstance cancels the active download or launch operation.
func (s *HomeService) CancelInstance(id string) error {
	if s.Registry.Get(id) == nil || !s.Registry.IsActive(id) {
		return NewNotFoundError("no active operation for instance")
	}
	s.Registry.Cancel(id)
	return nil
}

func (s *HomeService) LaunchInstance(id string) error {
	inst, err := s.Instances.Get(id)
	if err != nil {
		return NewNotFoundError("instance not found")
	}
	if inst.State != instances.StateReady && inst.State != instances.StateStopped {
		return NewConflictError("instance is not ready; install or repair it first")
	}
	detail, err := s.resolveInstanceDetail(context.Background(), inst)
	if err != nil {
		return NewUpstreamError(fmt.Sprintf("resolve metadata: %v", err))
	}
	settings := instances.EffectiveSettings(*inst, s.Defaults)
	javaPath := settings.JavaPath
	if javaPath == "" {
		found, scanErr := java.ScanJavaInstallations()
		if scanErr != nil {
			return NewIncompatibleError(scanErr.Error())
		}
		selected, selectErr := java.SelectJava(found, inst.MCVersion)
		if selectErr != nil {
			return NewIncompatibleError(selectErr.Error())
		}
		javaPath = selected.Path
	}
	nativesDir := filepath.Join(s.DataRoot, "instances", id, "natives")
	if err := os.RemoveAll(nativesDir); err != nil {
		return NewInternalError("clear natives: " + err.Error())
	}
	for _, library := range detail.Libraries {
		if !metadata.ShouldDownload(library.Rules, metadata.CurrentSystem()) {
			continue
		}
		nativePath, ok := metadata.ResolveNativePath(library, metadata.CurrentSystem())
		if !ok {
			continue
		}
		if err := launch.ExtractNatives(filepath.Join(s.DataRoot, nativePath), nativesDir); err != nil {
			return NewIntegrityError("extract natives: " + err.Error())
		}
	}
	launcher := s.Launch
	if launcher == nil {
		launcher = &LaunchService{Registry: s.Registry}
	}
	if s.Accounts == nil {
		return NewInternalError("account service is unavailable")
	}
	account, accessToken, err := s.Accounts.selectedAccount()
	if err != nil {
		return err
	}
	options := launch.Options{
		PlayerName:    account.Username,
		UUID:          account.UUID,
		AccessToken:   accessToken,
		UserType:      account.Type,
		VersionID:     id,
		GameDir:       filepath.Join(s.DataRoot, "instances", id, ".minecraft"),
		ClasspathRoot: s.DataRoot,
		AssetsDir:     filepath.Join(s.DataRoot, "assets"),
		NativesDir:    nativesDir,
		RamMB:         settings.MaxRamMB,
		Width:         settings.ResolutionW,
		Height:        settings.ResolutionH,
		JavaPath:      javaPath,
	}
	if account.Type == "ely.by" {
		injector, err := auth.EnsureAuthlibInjector(context.Background(), filepath.Join(s.DataRoot, "cache"), nil)
		if err != nil {
			return NewIntegrityError("authlib-injector: " + err.Error())
		}
		options.AuthlibInjector = injector
	}
	return launcher.Launch(*detail, options)
}

func (s *HomeService) StopInstance(id string) error {
	launcher := s.Launch
	if launcher == nil {
		launcher = &LaunchService{Registry: s.Registry}
	}
	return launcher.Stop(id)
}

func parseLoader(value string) (instances.LoaderType, error) {
	switch value {
	case "", "vanilla":
		return instances.LoaderVanilla, nil
	case "fabric":
		return instances.LoaderFabric, nil
	case "quilt":
		return instances.LoaderQuilt, nil
	default:
		return "", NewValidationError("invalid loader", "loader")
	}
}

func (s *HomeService) resolveInstanceDetail(ctx context.Context, inst *instances.Instance) (*metadata.VersionDetail, error) {
	client := metadata.NewClient(s.DataRoot)
	switch inst.Loader {
	case instances.LoaderFabric:
		return client.ResolveFabric(ctx, inst.MCVersion)
	case instances.LoaderQuilt:
		return client.ResolveQuilt(ctx, inst.MCVersion)
	default:
		return client.ResolveVersionChain(ctx, inst.MCVersion)
	}
}
