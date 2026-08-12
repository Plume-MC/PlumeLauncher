package services

import (
	"context"
	"fmt"
	"path/filepath"

	"plumelauncher/internal/auth"
	"plumelauncher/internal/downloader"
	"plumelauncher/internal/instances"
	"plumelauncher/internal/java"
	"plumelauncher/internal/launch"
	"plumelauncher/internal/metadata"
)

// HomeService exposes instance actions in terms of persisted instance IDs.
// The frontend never constructs artifact plans or launch command arguments.
type HomeService struct {
	DataRoot  string
	Defaults  instances.LauncherDefaults
	Instances *instances.Manager
	Registry  *instances.Registry
	Launch    *LaunchService
}

func (s *HomeService) ListInstances() ([]instances.Instance, error) {
	return s.Instances.List()
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

	client := metadata.NewClient(s.DataRoot)
	detail, err := client.ResolveVersionChain(context.Background(), inst.MCVersion)
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
	if err := orch.DownloadPlan(context.Background(), plan); err != nil {
		_ = s.Instances.UpdateState(id, instances.StateFailed)
		return err
	}
	if err := s.Instances.UpdateState(id, instances.StateVerifying); err != nil {
		return err
	}
	if err := s.Instances.UpdateState(id, instances.StateReady); err != nil {
		return err
	}
	return nil
}

func (s *HomeService) LaunchInstance(id string) error {
	inst, err := s.Instances.Get(id)
	if err != nil {
		return NewNotFoundError("instance not found")
	}
	detail, err := metadata.NewClient(s.DataRoot).ResolveVersionChain(context.Background(), inst.MCVersion)
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
	launcher := s.Launch
	if launcher == nil {
		launcher = &LaunchService{Registry: s.Registry}
	}
	return launcher.Launch(*detail, launch.Options{
		PlayerName:  "Player",
		UUID:        auth.OfflineUUID("Player"),
		AccessToken: "0",
		UserType:    "offline",
		VersionID:   id,
		GameDir:     filepath.Join(s.DataRoot, "instances", id, ".minecraft"),
		AssetsDir:   filepath.Join(s.DataRoot, "assets"),
		NativesDir:  filepath.Join(s.DataRoot, "instances", id, "natives"),
		RamMB:       settings.MaxRamMB,
		Width:       settings.ResolutionW,
		Height:      settings.ResolutionH,
		JavaPath:    javaPath,
	})
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
