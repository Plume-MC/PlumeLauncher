package instances

import (
	"fmt"
	"time"
)

// InstanceState represents the current state of an instance.
type InstanceState string

const (
	StateNotInstalled InstanceState = "not_installed"
	StatePlanning     InstanceState = "planning"
	StateDownloading  InstanceState = "downloading"
	StateVerifying    InstanceState = "verifying"
	StateReady        InstanceState = "ready"
	StateRunning      InstanceState = "running"
	StateStopped      InstanceState = "stopped"
	StateCrashed      InstanceState = "crashed"
	StateFailed       InstanceState = "failed"
)

// validTransitions defines allowed state transitions.
var validTransitions = map[InstanceState][]InstanceState{
	StateNotInstalled: {StatePlanning, StateFailed},
	StatePlanning:     {StateNotInstalled, StateDownloading, StateStopped, StateCrashed, StateFailed},
	StateDownloading:  {StateNotInstalled, StateVerifying, StateStopped, StateCrashed, StateFailed},
	StateVerifying:    {StateNotInstalled, StateReady, StateStopped, StateCrashed, StateFailed},
	StateReady:        {StateRunning},
	StateRunning:      {StateStopped, StateCrashed, StateFailed},
	StateStopped:      {StatePlanning},
	StateCrashed:      {StatePlanning},
	StateFailed:       {StatePlanning},
}

// CanTransition checks if a transition from one state to another is valid.
func CanTransition(from, to InstanceState) bool {
	allowed, ok := validTransitions[from]
	if !ok {
		return false
	}
	for _, s := range allowed {
		if s == to {
			return true
		}
	}
	return false
}

// Transition performs a state transition or returns an error.
func Transition(current InstanceState, next InstanceState) (InstanceState, error) {
	if !CanTransition(current, next) {
		return current, fmt.Errorf("invalid transition: %s → %s", current, next)
	}
	return next, nil
}

// Instance represents a Minecraft instance.
type Instance struct {
	ID            string        `json:"id"`
	Name          string        `json:"name"`
	MCVersion     string        `json:"mcVersion"`
	Loader        LoaderType    `json:"loader"`
	LoaderVersion string        `json:"loaderVersion,omitempty"`
	State         InstanceState `json:"state"`
	Settings      Settings      `json:"settings"`
	CreatedAt     time.Time     `json:"createdAt"`
	UpdatedAt     time.Time     `json:"updatedAt"`
}

// LoaderType identifies the mod loader.
type LoaderType string

const (
	LoaderVanilla LoaderType = "vanilla"
	LoaderFabric  LoaderType = "fabric"
	LoaderQuilt   LoaderType = "quilt"
)

// Settings holds per-instance launch settings.
type Settings struct {
	MinRamMB       *int    `json:"minRamMB,omitempty"`
	MaxRamMB       *int    `json:"maxRamMB,omitempty"`
	ResolutionW    *int    `json:"resolutionW,omitempty"`
	ResolutionH    *int    `json:"resolutionH,omitempty"`
	JavaPath       *string `json:"javaPath,omitempty"`
	JVMArgs        *string `json:"jvmArgs,omitempty"`
	WindowMode     *string `json:"windowMode,omitempty"`
	GPUPreference  *string `json:"gpuPreference,omitempty"`
	WrapperCommand *string `json:"wrapperCommand,omitempty"`
}
