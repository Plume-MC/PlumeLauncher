package services

import (
	"os/exec"
	"plumelauncher/internal/instances"
	"plumelauncher/internal/java"
	"plumelauncher/internal/launch"
	"plumelauncher/internal/metadata"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// LaunchService manages Minecraft launch.
type LaunchService struct {
	DataRoot  string
	Registry  *instances.Registry
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

	requiredMajor := java.RequiredJavaMajor(detail.ID)
	javaPath := opts.JavaPath
	if javaPath != "" {
		if _, err := java.ValidateJavaPath(javaPath, requiredMajor); err != nil {
			return NewIncompatibleError(err.Error())
		}
	} else {
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
	emit(s.App, EventLaunchState, LaunchStateEvent{InstanceID: opts.VersionID, State: "running"})
	s.mu.Lock()
	if s.processes == nil {
		s.processes = make(map[string]*exec.Cmd)
	}
	s.processes[opts.VersionID] = cmd
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.processes, opts.VersionID)
		s.mu.Unlock()
	}()

	// Monitor (blocks until exit)
	var stderr []string
	var stderrMu sync.Mutex
	err = launch.Monitor(cmd, func(line string, isStderr bool) {
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
	})

	s.Registry.Complete(opts.VersionID)

	if err != nil {
		emit(s.App, EventLaunchState, LaunchStateEvent{InstanceID: opts.VersionID, State: "failed"})
		if len(stderr) > 0 {
			return NewInternalError("game process error: " + launch.Redact(strings.Join(stderr, "\n"), opts.AccessToken))
		}
		return NewInternalError("game process error: " + err.Error())
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
