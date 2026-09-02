package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"plumelauncher/internal/auth"
	"plumelauncher/internal/instances"
	"plumelauncher/internal/java"
	"plumelauncher/internal/launch"
	"plumelauncher/internal/metadata"
)

func (s *HomeService) LaunchInstance(id string) error {
	inst, err := s.Instances.Get(id)
	if err != nil {
		return NewNotFoundError("instance not found")
	}
	if inst.State != instances.StateReady && inst.State != instances.StateStopped && inst.State != instances.StateCrashed {
		return NewConflictError("instance is not ready; install it first")
	}
	emit(s.App, EventLaunchState, LaunchStateEvent{InstanceID: id, State: "preparing"})
	detail, err := s.resolveInstanceDetail(context.Background(), inst)
	if err != nil {
		return NewUpstreamError(fmt.Sprintf("resolve metadata: %v", err))
	}
	plan := metadata.ResolvePlan(*detail, metadata.CurrentSystem())
	if err := s.ensureArtifacts(id, inst, plan); err != nil {
		return err
	}
	defaults, err := instances.LoadConfig(s.appStateRoot())
	if err != nil {
		return NewInternalError("load launcher settings: " + err.Error())
	}
	settings := instances.EffectiveSettings(*inst, defaults)
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
	if err := launch.ApplyGPUPreference(javaPath, settings.GPUPreference); err != nil && s.Logger != nil {
		s.Logger.Warn("apply GPU preference", "javaPath", javaPath, "preference", settings.GPUPreference, "error", err)
	}
	instanceDir, err := s.Instances.Dir(id)
	if err != nil {
		return NewNotFoundError("instance directory: " + err.Error())
	}
	nativesDir := filepath.Join(instanceDir, "natives")
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
		if err := launch.ExtractNatives(filepath.Join(s.DataRoot, metadata.ResolveLibraryDir(), nativePath), nativesDir, excludes); err != nil {
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
		GameDir:       filepath.Join(instanceDir, ".minecraft"),
		ClasspathRoot: s.DataRoot,
		AssetsDir:     filepath.Join(s.DataRoot, "assets"),
		NativesDir:    nativesDir,
		MinRamMB:      settings.MinRamMB,
		RamMB:         settings.MaxRamMB,
		Width:         settings.ResolutionW,
		Height:        settings.ResolutionH,
		WindowMode:    settings.WindowMode,
		JVMArgs:       launch.ParseArgumentString(settings.JVMArgs),
		Env:           launch.EnvironmentForGPU(settings.GPUPreference),
		JavaPath:      javaPath,
		JavaMajor:     java.RequiredJavaMajor(inst.MCVersion),
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
