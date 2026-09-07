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

func (s *LaunchService) HasRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.processes) > 0
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
	handedOff := false
	defer func() {
		if handedOff {
			return
		}
		if s.Registry.IsActive(opts.VersionID) {
			s.Registry.Fail(opts.VersionID)
		}
	}()

	// Build arguments
	args, err := launch.BuildArguments(detail, opts)
	if err != nil {
		return NewInternalError("failed to build arguments: " + err.Error())
	}

	// Find Java using the parent game version for loader metadata.
	javaVersion := javaVersionForLaunch(detail)
	requiredMajor := java.RequiredJavaMajor(javaVersion)
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
		selected, err := java.SelectJava(installs, javaVersion)
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
	handedOff = true
	go s.monitor(cmd, opts, op)
	return nil
}

func javaVersionForLaunch(detail metadata.VersionDetail) string {
	if detail.Jar != "" {
		return detail.Jar
	}
	return detail.ID
}

func (s *LaunchService) monitor(cmd *exec.Cmd, opts launch.Options, op *instances.Operation) {
	started := false
	defer func() {
		s.mu.Lock()
		delete(s.processes, opts.VersionID)
		s.mu.Unlock()
		if s.Registry.IsActive(opts.VersionID) {
			s.Registry.Fail(opts.VersionID)
		}
	}()

	var stderr []string
	var stderrMu sync.Mutex
	err := launch.MonitorWithStart(cmd, func(line string, isStderr bool) {
		level := launchLogLevel(line, isStderr)
		if isStderr {
			stderrMu.Lock()
			stderr = append(stderr, line)
			if len(stderr) > 20 {
				stderr = stderr[1:]
			}
			stderrMu.Unlock()
		}
		emit(s.App, EventLogLine, LogLineEvent{Level: level, Message: launch.Redact(line, opts.AccessToken), OperationID: op.ID, InstanceID: opts.VersionID})
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
			_ = transitionInstanceState(s.App, s.Instances, opts.VersionID, instances.StateRunning, op.ID)
		}
		emit(s.App, EventLaunchState, LaunchStateEvent{OperationID: op.ID, InstanceID: opts.VersionID, State: "running"})
	})

	if err != nil {
		launchError := launch.Redact(err.Error(), opts.AccessToken)
		if s.Logger != nil {
			s.Logger.Error("launch_failed", "instanceId", opts.VersionID, "operationId", op.ID, "error", launchError)
		}
		if !started {
			s.Registry.Fail(opts.VersionID)
			emit(s.App, EventLaunchState, LaunchStateEvent{OperationID: op.ID, InstanceID: opts.VersionID, State: "failed", Error: launchError})
			return
		}
		if current := s.Registry.Get(opts.VersionID); current != nil && current.Status == instances.OpStatusCancelled {
			if s.Instances != nil {
				_ = transitionInstanceState(s.App, s.Instances, opts.VersionID, instances.StateStopped, op.ID)
			}
			emit(s.App, EventLaunchState, LaunchStateEvent{OperationID: op.ID, InstanceID: opts.VersionID, State: "stopped"})
			return
		}
		var exitErr *launch.ProcessExitError
		if errors.As(err, &exitErr) {
			if s.Instances != nil {
				_ = transitionInstanceState(s.App, s.Instances, opts.VersionID, instances.StateCrashed, op.ID)
			}
			s.Registry.Fail(opts.VersionID)
			emit(s.App, EventLaunchState, LaunchStateEvent{OperationID: op.ID, InstanceID: opts.VersionID, State: "crashed", ExitCode: &exitErr.Code, Error: launchError})
		} else {
			if s.Instances != nil {
				_ = transitionInstanceState(s.App, s.Instances, opts.VersionID, instances.StateFailed, op.ID)
			}
			s.Registry.Fail(opts.VersionID)
			emit(s.App, EventLaunchState, LaunchStateEvent{OperationID: op.ID, InstanceID: opts.VersionID, State: "failed", Error: launchError})
		}
		if len(stderr) > 0 && s.Logger != nil {
			s.Logger.Error("game_process_error", "instanceId", opts.VersionID, "error", launch.Redact(strings.Join(stderr, "\n"), opts.AccessToken))
		}
		return
	}

	if s.Instances != nil {
		_ = transitionInstanceState(s.App, s.Instances, opts.VersionID, instances.StateStopped, op.ID)
	}
	s.Registry.Complete(opts.VersionID)
	if s.Logger != nil {
		s.Logger.Info("launch_stopped", "instanceId", opts.VersionID, "operationId", op.ID)
	}
	emit(s.App, EventLaunchState, LaunchStateEvent{OperationID: op.ID, InstanceID: opts.VersionID, State: "stopped"})
}

func launchLogLevel(line string, isStderr bool) string {
	upper := strings.ToUpper(line)
	switch {
	case strings.Contains(upper, "[ERROR]"), strings.Contains(upper, "[ERR]"), strings.Contains(upper, "ERROR:"), strings.Contains(upper, "FATAL"), strings.Contains(upper, "EXCEPTION"), strings.Contains(upper, "SEVERE"):
		return "error"
	case strings.Contains(upper, "WARN"):
		return "warning"
	case isStderr:
		return "warning"
	default:
		return "info"
	}
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
	emit(s.App, EventLaunchState, LaunchStateEvent{OperationID: op.ID, InstanceID: instanceID, State: "stopping"})
	if err := launch.Stop(cmd); err != nil {
		return NewInternalError("failed to stop launch: " + err.Error())
	}
	s.Registry.Cancel(instanceID)
	return nil
}

// KillAll forcefully terminates all running game processes.
// Called during application shutdown to prevent orphaned Java processes.
func (s *LaunchService) KillAll() {
	s.mu.Lock()
	procs := make(map[string]*exec.Cmd, len(s.processes))
	for k, v := range s.processes {
		procs[k] = v
	}
	s.mu.Unlock()

	for id, cmd := range procs {
		if cmd.Process != nil {
			if s.Logger != nil {
				s.Logger.Info("killing_instance", "instanceId", id, "pid", cmd.Process.Pid)
			}
			_ = cmd.Process.Kill()
		}
	}
}
