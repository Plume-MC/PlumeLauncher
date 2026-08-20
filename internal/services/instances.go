package services

import (
	"plumelauncher/internal/instances"
	"strings"
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
	if settings.MinRamMB != nil && settings.MaxRamMB != nil && *settings.MinRamMB > *settings.MaxRamMB {
		return NewValidationError("minRamMB must not exceed maxRamMB", "minRamMB", "maxRamMB")
	}
	if settings.GPUPreference != nil && !validGPUPreference(*settings.GPUPreference) {
		return NewValidationError("invalid GPU preference", "gpuPreference")
	}
	if settings.WrapperCommand != nil && !validWrapper(*settings.WrapperCommand) {
		return NewValidationError("invalid wrapper command", "wrapperCommand")
	}

	return s.Manager.UpdateSettings(id, settings)
}

// OpenInstanceFolder opens a selected instance through the platform file manager.
func (s *InstanceService) OpenInstanceFolder(id string) error {
	if id == "" {
		return NewValidationError("id is required", "id")
	}
	if err := instances.OpenInstanceFolder(s.DataRoot, id); err != nil {
		return NewNotFoundError(err.Error())
	}
	return nil
}

func validGPUPreference(value string) bool {
	return value == "" || value == "auto" || value == "discrete" || value == "integrated"
}

func validWrapper(value string) bool {
	return value == "" || (!strings.ContainsRune(value, 0) && len(strings.Fields(value)) > 0)
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
