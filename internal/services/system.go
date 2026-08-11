package services

import (
	"plumelauncher/internal/instances"
	"plumelauncher/internal/java"
	"strings"
)

// SystemService manages launcher settings and system operations.
type SystemService struct {
	DataRoot string
	Defaults instances.LauncherDefaults
}

// GetSettings returns the current launcher settings.
func (s *SystemService) GetSettings() instances.LauncherDefaults {
	return s.Defaults
}

// UpdateSettings updates the launcher settings.
func (s *SystemService) UpdateSettings(settings instances.LauncherDefaults) error {
	// Validate RAM bounds
	if settings.DefaultMinRamMB < 0 || settings.DefaultMinRamMB > 65536 {
		return NewValidationError("defaultMinRamMB out of range", "defaultMinRamMB")
	}
	if settings.DefaultMaxRamMB < 0 || settings.DefaultMaxRamMB > 65536 {
		return NewValidationError("defaultMaxRamMB out of range", "defaultMaxRamMB")
	}
	if settings.DefaultMinRamMB > settings.DefaultMaxRamMB {
		return NewValidationError("defaultMinRamMB must not exceed defaultMaxRamMB", "defaultMinRamMB", "defaultMaxRamMB")
	}
	if !validGPUPreference(settings.GPUPreference) {
		return NewValidationError("invalid GPU preference", "gpuPreference")
	}
	if !validWrapper(settings.WrapperCommand) {
		return NewValidationError("invalid wrapper command", "wrapperCommand")
	}
	if settings.DefaultJavaPath != "" {
		if _, err := java.CheckJava(settings.DefaultJavaPath); err != nil {
			return NewValidationError("default Java path is not executable", "defaultJavaPath")
		}
	}

	s.Defaults = settings

	// Persist
	return instances.SaveConfig(s.DataRoot, settings)
}

// WrapperArgs converts the validated persisted wrapper command to an argv prefix.
func WrapperArgs(value string) []string {
	return strings.Fields(value)
}

// GetDataRoot returns the data root path.
func (s *SystemService) GetDataRoot() string {
	return s.DataRoot
}

// ScanJava detects installed Java installations.
func (s *SystemService) ScanJava() ([]java.JavaInfo, error) {
	return java.ScanJavaInstallations()
}

// ValidateJavaPath checks if a Java path is valid and compatible.
func (s *SystemService) ValidateJavaPath(path string, requiredMajor int) (*java.JavaInfo, error) {
	if path == "" {
		return nil, NewValidationError("path is required", "path")
	}
	return java.ValidateJavaPath(path, requiredMajor)
}
