package services

import (
	"plumelauncher/internal/instances"
)

// InstanceService manages Minecraft instances.
type InstanceService struct {
	DataRoot string
	Manager  *instances.Manager
}

// CreateInstance creates a new instance.
func (s *InstanceService) CreateInstance(name, mcVersion string, loader string) (*instances.Instance, error) {
	if name == "" {
		return nil, NewValidationError("name is required", "name")
	}
	if mcVersion == "" {
		return nil, NewValidationError("mcVersion is required", "mcVersion")
	}

	var loaderType instances.LoaderType
	switch loader {
	case "vanilla", "":
		loaderType = instances.LoaderVanilla
	case "fabric":
		loaderType = instances.LoaderFabric
	case "quilt":
		loaderType = instances.LoaderQuilt
	default:
		return nil, NewValidationError("invalid loader", "loader")
	}

	inst, err := s.Manager.Create(name, mcVersion, loaderType)
	if err != nil {
		return nil, NewInternalError(err.Error())
	}
	return inst, nil
}

// ListInstances returns all instances.
func (s *InstanceService) ListInstances() ([]instances.Instance, error) {
	return s.Manager.List()
}

// GetInstance returns an instance by ID.
func (s *InstanceService) GetInstance(id string) (*instances.Instance, error) {
	if id == "" {
		return nil, NewValidationError("id is required", "id")
	}

	inst, err := s.Manager.Get(id)
	if err != nil {
		return nil, NewNotFoundError("instance not found: " + id)
	}
	return inst, nil
}

// UpdateInstanceSettings updates instance settings.
func (s *InstanceService) UpdateInstanceSettings(id string, settings instances.Settings) error {
	if id == "" {
		return NewValidationError("id is required", "id")
	}

	// Validate RAM bounds
	if settings.MinRamMB != nil && *settings.MinRamMB < 256 {
		return NewValidationError("minRamMB must be at least 256", "minRamMB")
	}
	if settings.MaxRamMB != nil && *settings.MaxRamMB > 65536 {
		return NewValidationError("maxRamMB must be at most 65536", "maxRamMB")
	}

	return s.Manager.UpdateSettings(id, settings)
}

// DeleteInstance removes an instance.
func (s *InstanceService) DeleteInstance(id string) error {
	if id == "" {
		return NewValidationError("id is required", "id")
	}

	err := s.Manager.Delete(id)
	if err != nil {
		return NewNotFoundError("instance not found: " + id)
	}
	return nil
}

// TransitionState performs a state transition on an instance.
func (s *InstanceService) TransitionState(id string, newState string) error {
	if id == "" {
		return NewValidationError("id is required", "id")
	}

	state := instances.InstanceState(newState)
	if !instances.CanTransition("", state) && newState != string(instances.StateNotInstalled) {
		// Validate target state exists
		switch state {
		case instances.StatePlanning, instances.StateDownloading, instances.StateVerifying,
			instances.StateReady, instances.StateRunning, instances.StateStopped,
			instances.StateCrashed, instances.StateFailed:
			// valid target states
		default:
			return NewValidationError("invalid state: "+newState, "state")
		}
	}

	return s.Manager.UpdateState(id, state)
}
