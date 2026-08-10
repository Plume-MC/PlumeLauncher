package services

import (
	"plumelauncher/internal/instances"
	"plumelauncher/internal/java"
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

	s.Defaults = settings

	// Persist
	return instances.SaveConfig(s.DataRoot, settings)
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
