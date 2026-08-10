package services

import (
	"plumelauncher/internal/instances"
	"plumelauncher/internal/java"
	"plumelauncher/internal/launch"
	"plumelauncher/internal/metadata"
)

// LaunchService manages Minecraft launch.
type LaunchService struct {
	DataRoot string
	Registry *instances.Registry
}

// Launch starts a Minecraft instance.
func (s *LaunchService) Launch(detail metadata.VersionDetail, opts launch.Options) error {
	// Check if operation already active
	if s.Registry.IsActive(opts.VersionID) {
		return NewConflictError("launch already active for " + opts.VersionID)
	}

	// Start operation
	op, err := s.Registry.Start(opts.VersionID, instances.OpLaunch)
	if err != nil {
		return NewConflictError(err.Error())
	}
	defer func() {
		if op.Status == instances.OpStatusRunning {
			s.Registry.Fail(opts.VersionID)
		}
	}()

	// Build arguments
	args, err := launch.BuildArguments(detail, opts)
	if err != nil {
		return NewInternalError("failed to build arguments: " + err.Error())
	}

	// Find Java
	installs, err := java.ScanJavaInstallations()
	if err != nil || len(installs) == 0 {
		return NewIncompatibleError("no Java installation found")
	}

	javaPath := opts.JavaPath
	if javaPath == "" {
		javaPath = installs[0].Path
	}

	// Launch process
	cmd, err := launch.Launch(javaPath, args, opts.GameDir, nil)
	if err != nil {
		return NewInternalError("failed to launch: " + err.Error())
	}

	// Monitor (blocks until exit)
	err = launch.Monitor(cmd, func(line string, isStderr bool) {
		// Log callback — could emit Wails event here
	})

	s.Registry.Complete(opts.VersionID)

	if err != nil {
		return NewInternalError("game process error: " + err.Error())
	}

	return nil
}

// Stop stops a running instance.
func (s *LaunchService) Stop(instanceID string) error {
	op := s.Registry.Get(instanceID)
	if op == nil || op.Status != instances.OpStatusRunning {
		return NewNotFoundError("no active launch for " + instanceID)
	}

	s.Registry.Cancel(instanceID)
	return nil
}
