package services

import (
	"errors"
	"os/exec"
	"plumelauncher/internal/instances"
	"plumelauncher/internal/java"
	"plumelauncher/internal/launch"
	"plumelauncher/internal/logging"
	"plumelauncher/internal/metadata"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// LaunchService manages Minecraft launch.
type LaunchService struct {
	DataRoot  string
	Registry  *instances.Registry
	Instances *instances.Manager
	Logger    *logging.Logger
	App       *application.App
	mu        sync.Mutex
	processes map[string]*exec.Cmd
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
	if s.Logger != nil {
		s.Logger.AddSecrets(opts.AccessToken)
		s.Logger.Info("launch_requested", "instanceId", opts.VersionID, "operationId", op.ID)
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
	requiredMajor := java.RequiredJavaMajor(detail.ID)
	javaPath := opts.JavaPath
	if javaPath != "" {
		if _, err := java.ValidateJavaPath(javaPath, requiredMajor); err != nil {
			return NewIncompatibleError(err.Error())
		}
	} else {
		installs, err := java.ScanJavaInstallations()
		if err != nil || len(installs) == 0 {
			return NewIncompatibleError("no Java installation found")
		}
		selected, err := java.SelectJava(installs, detail.ID)
		if err != nil {
			return NewIncompatibleError(err.Error())
		}
		javaPath = selected.Path
	}

	// Launch process
	command, commandArgs, err := launch.BuildCommand(javaPath, args, opts.Wrapper)
	if err != nil {
		return NewValidationError(err.Error(), "wrapper")
	}
	cmd, err := launch.Launch(command, commandArgs, opts.GameDir, opts.Env)
	if err != nil {
		return NewInternalError("failed to launch: " + err.Error())
	}
	started := false
	defer func() {
		s.mu.Lock()
		if s.processes != nil {
			delete(s.processes, opts.VersionID)
		}
		s.mu.Unlock()
	}()

	// Monitor (blocks until exit)
	var stderr []string
	var stderrMu sync.Mutex
	err = launch.MonitorWithStart(cmd, func(line string, isStderr bool) {
		level := "info"
		if isStderr {
			level = "error"
			stderrMu.Lock()
			stderr = append(stderr, line)
			if len(stderr) > 20 {
				stderr = stderr[1:]
			}
			stderrMu.Unlock()
		}
		emit(s.App, EventLogLine, LogLineEvent{Level: level, Message: launch.Redact(line, opts.AccessToken), InstanceID: opts.VersionID})
		if s.Logger != nil {
			s.Logger.Info("game_output", "instanceId", opts.VersionID, "level", level, "message", line)
		}
	}, func() {
		started = true
		s.mu.Lock()
		if s.processes == nil {
			s.processes = make(map[string]*exec.Cmd)
		}
		s.processes[opts.VersionID] = cmd
		s.mu.Unlock()
		if s.Instances != nil {
			_ = s.Instances.UpdateState(opts.VersionID, instances.StateRunning)
		}
		emit(s.App, EventLaunchState, LaunchStateEvent{InstanceID: opts.VersionID, State: "running"})
	})

	if err != nil {
		if s.Logger != nil {
			s.Logger.Error("launch_failed", "instanceId", opts.VersionID, "error", launch.Redact(err.Error(), opts.AccessToken))
		}
		if !started {
			s.Registry.Fail(opts.VersionID)
			emit(s.App, EventLaunchState, LaunchStateEvent{InstanceID: opts.VersionID, State: "failed"})
			return NewInternalError("failed to start game: " + err.Error())
		}
		if op := s.Registry.Get(opts.VersionID); op != nil && op.Status == instances.OpStatusCancelled {
			if s.Instances != nil {
				_ = s.Instances.UpdateState(opts.VersionID, instances.StateStopped)
			}
			emit(s.App, EventLaunchState, LaunchStateEvent{InstanceID: opts.VersionID, State: "stopped"})
			return nil
		}
		var exitErr *launch.ProcessExitError
		if errors.As(err, &exitErr) {
			if s.Instances != nil {
				_ = s.Instances.UpdateState(opts.VersionID, instances.StateCrashed)
			}
			s.Registry.Fail(opts.VersionID)
			emit(s.App, EventLaunchState, LaunchStateEvent{InstanceID: opts.VersionID, State: "crashed", ExitCode: &exitErr.Code})
		} else {
			if s.Instances != nil {
				_ = s.Instances.UpdateState(opts.VersionID, instances.StateFailed)
			}
			s.Registry.Fail(opts.VersionID)
			emit(s.App, EventLaunchState, LaunchStateEvent{InstanceID: opts.VersionID, State: "failed"})
		}
		if len(stderr) > 0 {
			return NewInternalError("game process error: " + launch.Redact(strings.Join(stderr, "\n"), opts.AccessToken))
		}
		return NewInternalError("game process error: " + err.Error())
	}

	if s.Instances != nil {
		_ = s.Instances.UpdateState(opts.VersionID, instances.StateStopped)
	}
	s.Registry.Complete(opts.VersionID)
	if s.Logger != nil {
		s.Logger.Info("launch_stopped", "instanceId", opts.VersionID, "operationId", op.ID)
	}
	emit(s.App, EventLaunchState, LaunchStateEvent{InstanceID: opts.VersionID, State: "stopped"})
	return nil
}

// Stop stops a running instance.
func (s *LaunchService) Stop(instanceID string) error {
	op := s.Registry.Get(instanceID)
	if op == nil || op.Status != instances.OpStatusRunning {
		return NewNotFoundError("no active launch for " + instanceID)
	}

	s.mu.Lock()
	cmd := s.processes[instanceID]
	s.mu.Unlock()
	if cmd == nil {
		return NewNotFoundError("no process for active launch")
	}
	if err := launch.Stop(cmd); err != nil {
		return NewInternalError("failed to stop launch: " + err.Error())
	}
	s.Registry.Cancel(instanceID)
	return nil
}
