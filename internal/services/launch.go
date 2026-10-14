package services

import (
	"errors"
	"os/exec"
	"plumelauncher/internal/instances"
	"plumelauncher/internal/java"
	"plumelauncher/internal/launch"
	"plumelauncher/internal/logging"
	"plumelauncher/internal/metadata"
	"plumelauncher/internal/presence"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// LaunchService manages Minecraft launch.
type LaunchService struct {
	DataRoot  string
	Registry  *instances.Registry
	Instances *instances.Manager
	Logger    *logging.Logger
	App       *application.App
	Presence  *presence.Client
	mu        sync.Mutex
	processes map[string]*exec.Cmd
}

func (s *LaunchService) HasRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.processes) > 0
}

// launchRequiredMajor resolves the Java major a launch needs. Mojang's
// javaVersion field wins when present; loader detail IDs (e.g.
// "fabric-loader-0.16.14-1.20.1") do not parse as versions, so the plain
// Minecraft version is used as fallback instead of detail.ID.
func launchRequiredMajor(detail metadata.VersionDetail, mcVersion string) int {
	if detail.JavaVersion.MajorVersion > 0 {
		return detail.JavaVersion.MajorVersion
	}
	if mcVersion == "" {
		mcVersion = detail.ID
	}
	return java.RequiredJavaMajor(mcVersion)
}

// Launch starts a Minecraft instance.
func (s *LaunchService) Launch(detail metadata.VersionDetail, opts launch.Options) error {
	// Check if operation already active
	if s.Registry.IsActive(opts.VersionID) {
		return NewConflictError("Minecraft is already running for this instance.")
	}

	// Start operation
	op, err := s.Registry.Start(opts.VersionID, instances.OpLaunch)
	if err != nil {
		return NewConflictError("Minecraft is already running for this instance.")
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
		return NewInternalError("Unable to prepare the launch. Try again.")
	}

	// Find Java
	mcVersion := opts.MCVersion
	if mcVersion == "" {
		mcVersion = detail.ID
	}
	requiredMajor := launchRequiredMajor(detail, opts.MCVersion)
	javaPath := opts.JavaPath
	if javaPath != "" {
		if _, err := java.ValidateJavaPath(javaPath, requiredMajor); err != nil {
			return NewIncompatibleError("The configured Java does not work for this Minecraft version. Pick another Java in Settings.")
		}
	} else {
		installs, err := java.ScanJavaInstallations()
		if err != nil || len(installs) == 0 {
			return NewIncompatibleError("No Java installation found. Download one from Settings > Java.")
		}
		selected, err := java.SelectJava(installs, mcVersion)
		if err != nil {
			return NewIncompatibleError("Unable to pick a Java installation. Download one from Settings > Java.")
		}
		javaPath = selected.Path
	}

	// Launch process
	command, commandArgs, err := launch.BuildCommand(javaPath, args, opts.Wrapper)
	if err != nil {
		return NewValidationError("The wrapper command is invalid. Check Settings for the correct format.", "wrapper")
	}
	cmd, err := launch.Launch(command, commandArgs, opts.GameDir, opts.Env)
	if err != nil {
		return NewInternalError("Unable to start Minecraft.")
	}
	handedOff = true
	go s.monitor(cmd, opts, op)
	return nil
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
		// Every monitor exit path (stopped/crashed/failed) lands here.
		s.clearPresence()
	}()

	var stderr []string
	var stderrMu sync.Mutex
	if op.CancelContext.Err() != nil {
		emit(s.App, EventLaunchState, LaunchStateEvent{OperationID: op.ID, InstanceID: opts.VersionID, State: "stopped"})
		return
	}
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
		if op.CancelContext.Err() != nil {
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
			return
		}
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
		// Presence is best-effort and async: an unreachable Discord must
		// never delay or fail the launch monitor.
		s.publishPresence(cmd.Process.Pid, opts.VersionID)
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

// publishPresence sends one activity update for a running game. It runs in
// its own goroutine and swallows errors; presence is strictly optional.
func (s *LaunchService) publishPresence(pid int, instanceID string) {
	if s.Presence == nil || s.Instances == nil {
		return
	}
	presenceClient := s.Presence
	go func() {
		inst, err := s.Instances.Get(instanceID)
		if err != nil || inst == nil {
			return
		}
		defaults, err := instances.LoadConfig(s.DataRoot)
		if err != nil {
			return
		}
		loader := string(inst.Loader)
		if loader == "vanilla" {
			loader = "" // vanilla has no loader art
		}
		info := presence.InstanceInfo{Name: inst.Name, MCVersion: inst.MCVersion, Loader: loader}
		activity := presence.BuildActivity(pid, info, defaults, time.Now())
		if activity == nil {
			return
		}
		_ = presenceClient.SetActivity(activity)
	}()
}

// clearPresence removes the published activity when the game exits.
func (s *LaunchService) clearPresence() {
	if s.Presence == nil {
		return
	}
	_ = s.Presence.SetActivity(nil)
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
		return NewNotFoundError("No active game for this instance.")
	}

	s.mu.Lock()
	cmd := s.processes[instanceID]
	s.mu.Unlock()
	if cmd == nil {
		emit(s.App, EventLaunchState, LaunchStateEvent{OperationID: op.ID, InstanceID: instanceID, State: "stopping"})
		s.Registry.Cancel(instanceID)
		return nil
	}
	emit(s.App, EventLaunchState, LaunchStateEvent{OperationID: op.ID, InstanceID: instanceID, State: "stopping"})
	if err := launch.Stop(cmd); err != nil {
		return NewInternalError("Unable to stop Minecraft.")
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
