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

func (s *HomeService) LaunchInstance(id string) (err error) {
	inst, err := s.Instances.Get(id)
	if err != nil {
		return NewNotFoundError("Instance not found.")
	}
	if inst.State != instances.StateReady && inst.State != instances.StateStopped && inst.State != instances.StateCrashed {
		return NewConflictError("This instance is not ready. Install it first.")
	}
	emit(s.App, EventLaunchState, LaunchStateEvent{InstanceID: id, State: "preparing"})
	defer func() {
		if err != nil {
			emit(s.App, EventLaunchState, LaunchStateEvent{InstanceID: id, State: "failed", Error: err.Error()})
		}
	}()
	detail, err := s.resolveInstanceDetail(context.Background(), inst)
	if err != nil {
		return NewUpstreamError("Unable to load version details. Check your connection and try again.")
	}
	plan := metadata.ResolvePlan(*detail, metadata.CurrentSystem())
	if err := s.ensureArtifacts(id, inst, plan); err != nil {
		return err
	}
	defaults, err := instances.LoadConfig(s.DataRoot)
	if err != nil {
		return NewInternalError("Unable to load launcher settings.")
	}
	settings := instances.EffectiveSettings(*inst, defaults)
	javaPath := settings.JavaPath
	if javaPath == "" {
		found, scanErr := java.ScanJavaInstallations()
		if scanErr != nil {
			return NewIncompatibleError("Unable to scan for Java installations.")
		}
		selected, selectErr := java.SelectJava(found, inst.MCVersion)
		if selectErr != nil {
			required := java.RequiredJavaMajor(inst.MCVersion)
			return NewIncompatibleError(fmt.Sprintf(
				"Java %d is required for Minecraft %s but was not found. Download it from Settings > Java, or install Java %d manually.",
				required, inst.MCVersion, required,
			))
		}
		javaPath = selected.Path
	}
	if err := launch.ApplyGPUPreference(javaPath, settings.GPUPreference); err != nil && s.Logger != nil {
		s.Logger.Warn("apply GPU preference", "javaPath", javaPath, "preference", settings.GPUPreference, "error", err)
	}
	instanceDir, err := s.Instances.Dir(id)
	if err != nil {
		return NewNotFoundError("Unable to open the instance folder.")
	}
	nativesDir := filepath.Join(instanceDir, "natives")
	if err := os.RemoveAll(nativesDir); err != nil {
		return NewInternalError("Unable to prepare the instance folder.")
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
			return NewIntegrityError("Unable to prepare game files. Try repairing the instance.")
		}
	}
	launcher := s.Launch
	if launcher == nil {
		launcher = &LaunchService{Registry: s.Registry}
	}
	if s.Accounts == nil {
		return NewInternalError("Something went wrong. Restart the launcher and try again.")
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
		return NewValidationError("The wrapper command is invalid. Check Settings for the correct format.", "wrapper")
	}
	if needsAuthlibInjector(account.Type) {
		injector, err := auth.EnsureAuthlibInjector(context.Background(), filepath.Join(s.DataRoot, "cache"), nil)
		if err != nil {
			return NewIntegrityError("Unable to set up Ely.by login. Try repairing the instance.")
		}
		options.AuthlibInjector = injector
	}
	return launcher.Launch(*detail, options)
}

// needsAuthlibInjector reports whether the account type requires the
// Ely.by authlib-injector agent. Only Ely.by uses runtime injection;
// offline and Microsoft sessions launch without a Java agent.
func needsAuthlibInjector(accountType string) bool {
	return accountType == AccountTypeElyBy
}

func (s *HomeService) StopInstance(id string) error {
	launcher := s.Launch
	if launcher == nil {
		launcher = &LaunchService{Registry: s.Registry}
	}
	return launcher.Stop(id)
}
