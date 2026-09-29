package services

import (
	"plumelauncher/internal/instances"
	"plumelauncher/internal/launch"
)

// InstanceService manages Minecraft instances.
type InstanceService struct {
	DataRoot string
	Manager  *instances.Manager
}

// CreateInstance creates a new instance.
func (s *InstanceService) CreateInstance(name, mcVersion string, loader string) (*instances.Instance, error) {
	if name == "" {
		return nil, NewValidationError("Enter an instance name.", "name")
	}
	if mcVersion == "" {
		return nil, NewValidationError("Pick a Minecraft version.", "mcVersion")
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
		return nil, NewValidationError("Pick a valid loader.", "loader")
	}

	inst, err := s.Manager.Create(name, mcVersion, loaderType)
	if err != nil {
		return nil, NewInternalError("Unable to create the instance.")
	}
	return inst, nil
}

// ListInstances returns all instances.
func (s *InstanceService) ListInstances() ([]instances.Instance, error) {
	items, err := s.Manager.List()
	if err != nil {
		return nil, NewInternalError("Unable to list instances.")
	}
	return items, nil
}

// GetInstance returns an instance by ID.
func (s *InstanceService) GetInstance(id string) (*instances.Instance, error) {
	if id == "" {
		return nil, NewValidationError("Something went wrong. Restart the launcher and try again.", "id")
	}

	inst, err := s.Manager.Get(id)
	if err != nil {
		return nil, NewNotFoundError("Instance not found.")
	}
	return inst, nil
}

// UpdateInstanceSettings updates instance settings.
func (s *InstanceService) UpdateInstanceSettings(id string, settings instances.Settings) error {
	if id == "" {
		return NewValidationError("Something went wrong. Restart the launcher and try again.", "id")
	}

	// Validate RAM bounds
	if settings.MinRamMB != nil && *settings.MinRamMB < 256 {
		return NewValidationError("Minimum RAM must be at least 256 MB.", "minRamMB")
	}
	if settings.MaxRamMB != nil && *settings.MaxRamMB > 65536 {
		return NewValidationError("Maximum RAM must be at most 65536 MB.", "maxRamMB")
	}
	if settings.MinRamMB != nil && settings.MaxRamMB != nil && *settings.MinRamMB > *settings.MaxRamMB {
		return NewValidationError("Minimum RAM must not exceed maximum RAM.", "minRamMB", "maxRamMB")
	}
	if settings.GPUPreference != nil && !validGPUPreference(*settings.GPUPreference) {
		return NewValidationError("Pick a valid graphics option.", "gpuPreference")
	}
	if settings.WrapperCommand != nil && !validWrapper(*settings.WrapperCommand) {
		return NewValidationError("The wrapper command is invalid. Check Settings for the correct format.", "wrapperCommand")
	}

	if _, err := s.Manager.Get(id); err != nil {
		return NewNotFoundError("Instance not found.")
	}
	if err := s.Manager.UpdateSettings(id, settings); err != nil {
		return NewInternalError("Unable to save instance settings.")
	}
	return nil
}

// OpenInstanceFolder opens a selected instance through the platform file manager.
func (s *InstanceService) OpenInstanceFolder(id string) error {
	if id == "" {
		return NewValidationError("Something went wrong. Restart the launcher and try again.", "id")
	}
	if err := instances.OpenInstanceFolder(s.DataRoot, id); err != nil {
		return NewNotFoundError("Unable to open the instance folder.")
	}
	return nil
}

func validGPUPreference(value string) bool {
	return value == "" || value == "auto" || value == "discrete" || value == "integrated"
}

func validWrapper(value string) bool {
	_, err := launch.ParseAndValidateWrapper(value)
	return err == nil
}

// DeleteInstance removes an instance.
func (s *InstanceService) DeleteInstance(id string) error {
	if id == "" {
		return NewValidationError("Something went wrong. Restart the launcher and try again.", "id")
	}

	err := s.Manager.Delete(id)
	if err != nil {
		return NewNotFoundError("Instance not found.")
	}
	return nil
}
