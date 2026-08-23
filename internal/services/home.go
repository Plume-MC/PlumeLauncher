package services

import (
	"context"
	"errors"
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
	"plumelauncher/internal/security"

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

func (s *HomeService) ListInstances() ([]instances.Instance, error) {
	return s.Instances.List()
}

// SupportedVersions returns stable releases listed by the selected metadata source.
func (s *HomeService) SupportedVersions(loader string) ([]string, error) {
	client := metadata.NewClient(s.DataRoot)
	if loader == "" || loader == "vanilla" {
		manifest, err := client.FetchManifest(context.Background())
		if err != nil {
			return nil, NewUpstreamError(fmt.Sprintf("resolve vanilla metadata: %v", err))
		}
		versions := make([]string, 0, len(manifest.Versions))
		for _, version := range manifest.Versions {
			if version.Type == "release" {
				versions = append(versions, version.ID)
			}
		}
		return versions, nil
	}
	if loader != "fabric" && loader != "quilt" {
		return nil, NewValidationError("invalid loader", "loader")
	}
	versions, err := client.SupportedLoaderVersions(context.Background(), loader)
	if err != nil {
		return nil, NewUpstreamError(fmt.Sprintf("resolve %s metadata: %v", loader, err))
	}
	return versions, nil
}

// LoaderVersions returns versions of the selected Fabric or Quilt loader.
func (s *HomeService) LoaderVersions(loader, gameVersion string) ([]string, error) {
	if loader != "fabric" && loader != "quilt" {
		return nil, NewValidationError("loader versions are unavailable for this loader", "loader")
	}
	versions, err := metadata.NewClient(s.DataRoot).LoaderVersions(context.Background(), loader, gameVersion)
	if err != nil {
		return nil, NewUpstreamError(fmt.Sprintf("resolve %s loader versions: %v", loader, err))
	}
	return versions, nil
}

func (s *HomeService) CreateInstance(name, version, loader, loaderVersion string) (*instances.Instance, error) {
	loaderType, err := parseLoader(loader)
	if err != nil {
		return nil, err
	}
	if loaderType != instances.LoaderVanilla && loaderVersion == "" {
		return nil, NewValidationError("loaderVersion is required", "loaderVersion")
	}
	return s.Instances.CreateWithLoaderVersion(name, version, loaderType, loaderVersion)
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
	for _, artifact := range plan.Artifacts {
		if err := security.ValidateArtifactURL(artifact.URL); err != nil {
			return NewValidationError(err.Error(), "artifact.url")
		}
	}
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
		status := "failed"
		if errors.Is(err, context.Canceled) || s.Registry.Get(id).Status == instances.OpStatusCancelled {
			status = "cancelled"
		}
		emit(s.App, EventDownloadProgress, DownloadProgressEvent{OperationID: op.ID, InstanceID: id, Status: status, Error: err.Error()})
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

// RetryInstance retries installation or repair for a failed instance.
func (s *HomeService) RetryInstance(id string) error {
	inst, err := s.Instances.Get(id)
	if err != nil {
		return NewNotFoundError("instance not found")
	}
	if inst.State != instances.StateFailed && inst.State != instances.StateCrashed {
		return NewConflictError("instance has no failed operation to retry")
	}
	return s.InstallInstance(id)
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
		excludes := []string(nil)
		if library.Extract != nil {
			excludes = library.Extract.Exclude
		}
		if err := launch.ExtractNatives(filepath.Join(s.DataRoot, nativePath), nativesDir, excludes); err != nil {
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
		MinRamMB:      settings.MinRamMB,
		RamMB:         settings.MaxRamMB,
		Width:         settings.ResolutionW,
		Height:        settings.ResolutionH,
		WindowMode:    settings.WindowMode,
		GPU:           settings.GPUPreference,
		JVMArgs:       launch.ParseArgumentString(settings.JVMArgs),
		Env:           launch.EnvironmentForGPU(settings.GPUPreference),
		JavaPath:      javaPath,
	}
	options.Wrapper, err = launch.ParseAndValidateWrapper(settings.WrapperCommand)
	if err != nil {
		return NewValidationError(err.Error(), "wrapper")
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
	var detail *metadata.VersionDetail
	var err error
	switch inst.Loader {
	case instances.LoaderFabric:
		detail, err = client.ResolveFabric(ctx, inst.MCVersion, inst.LoaderVersion)
	case instances.LoaderQuilt:
		detail, err = client.ResolveQuilt(ctx, inst.MCVersion, inst.LoaderVersion)
	default:
		detail, err = client.ResolveVersionChain(ctx, inst.MCVersion)
	}
	if err != nil || detail == nil || detail.AssetIndex.URL == "" {
		return detail, err
	}
	objects, err := client.FetchAssetObjects(ctx, detail.AssetIndex)
	if err != nil {
		return nil, err
	}
	detail.AssetIndex.Objects = objects
	return detail, nil
}
