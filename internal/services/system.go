package services

import (
	"os"
	"path/filepath"

	"plumelauncher/internal/bootstrap"
	"plumelauncher/internal/instances"
	"plumelauncher/internal/java"
	"plumelauncher/internal/launch"
	"plumelauncher/internal/platform"
)

// SystemService manages launcher settings and system operations.
type SystemService struct {
	DataRoot string // canonical launcher data root
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
		if _, err := java.ValidateJavaPath(settings.DefaultJavaPath, 0); err != nil {
			return NewValidationError("default Java path is not executable", "defaultJavaPath")
		}
	}

	if err := instances.SaveConfig(s.DataRoot, settings); err != nil {
		return NewInternalError("save launcher settings: " + err.Error())
	}
	s.Defaults = settings
	return nil
}

// WrapperArgs converts the validated persisted wrapper command to an argv prefix.
func WrapperArgs(value string) []string {
	args, err := launch.ParseAndValidateWrapper(value)
	if err != nil {
		return nil
	}
	return args
}

// GetDataRoot returns the canonical launcher data root.
func (s *SystemService) GetDataRoot() string {
	return s.DataRoot
}

// GetAppRoot returns the canonical data root for older clients.
func (s *SystemService) GetAppRoot() string {
	return s.DataRoot
}

// IsPortableMode reports whether launcher data is stored beside the executable.
func (s *SystemService) IsPortableMode() bool {
	return bootstrap.IsPortableMode(s.DataRoot)
}

// UpdateDataRoot stores the canonical data root to use after the launcher restarts.
func (s *SystemService) UpdateDataRoot(path string) error {
	if path == "" {
		return NewValidationError("data root is required", "dataRoot")
	}
	if err := bootstrap.SetDataRoot(path); err != nil {
		return NewValidationError("invalid data root: "+err.Error(), "dataRoot")
	}
	return nil
}

// OpenLogFolder opens the canonical data root's log directory.
func (s *SystemService) OpenLogFolder() error {
	logDir := filepath.Join(s.DataRoot, "logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return NewInternalError("create log folder: " + err.Error())
	}
	if err := platform.OpenFileManager(logDir); err != nil {
		return NewInternalError("open log folder: " + err.Error())
	}
	return nil
}

func (s *SystemService) OpenAppRoot() error {
	return s.openFolder(s.DataRoot, "data")
}

func (s *SystemService) OpenGameRoot() error {
	return s.openFolder(s.DataRoot, "game data")
}

func (s *SystemService) openFolder(path, label string) error {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return NewInternalError("create " + label + " folder: " + err.Error())
	}
	if err := platform.OpenFileManager(path); err != nil {
		return NewInternalError("open " + label + " folder: " + err.Error())
	}
	return nil
}

// ScanJava detects installed Java installations.
func (s *SystemService) ScanJava() ([]java.JavaInfo, error) {
	installed, err := java.RescanJavaInstallations()
	if err != nil {
		return nil, NewInternalError("scan Java installations: " + err.Error())
	}
	return installed, nil
}

// JavaRuntimes returns all verified runtimes, marking the PATH runtime and custom entries.
func (s *SystemService) JavaRuntimes() ([]java.JavaInfo, error) {
	installed, err := java.ScanJavaInstallations()
	if err != nil {
		return nil, NewInternalError("scan Java installations: " + err.Error())
	}
	for i := range installed {
		installed[i].Source = "Detected"
	}
	if system, err := java.SystemDefault(); err == nil && system != nil {
		installed = append([]java.JavaInfo{*system}, installed...)
	}
	for _, path := range s.Defaults.CustomJavaPaths {
		if info, err := java.ValidateJavaPath(path, 0); err == nil {
			info.Source = "Custom"
			installed = append(installed, *info)
		}
	}
	seen := make(map[string]bool)
	runtimes := make([]java.JavaInfo, 0, len(installed))
	for _, info := range installed {
		path, err := filepath.EvalSymlinks(info.Path)
		if err == nil {
			info.Path = path
		}
		if !seen[info.Path] {
			seen[info.Path] = true
			runtimes = append(runtimes, info)
		}
	}
	return runtimes, nil
}

// AddCustomJava verifies a Java executable before persisting it as a custom runtime.
func (s *SystemService) AddCustomJava(path string) (*java.JavaInfo, error) {
	info, err := java.ValidateJavaPath(path, 0)
	if err != nil {
		return nil, NewValidationError(err.Error(), "javaPath")
	}
	for _, existing := range s.Defaults.CustomJavaPaths {
		if existing == info.Path {
			info.Source = "Custom"
			return info, nil
		}
	}
	next := s.Defaults
	next.CustomJavaPaths = append(append([]string(nil), s.Defaults.CustomJavaPaths...), info.Path)
	if err := instances.SaveConfig(s.DataRoot, next); err != nil {
		return nil, NewInternalError("save custom Java: " + err.Error())
	}
	s.Defaults = next
	info.Source = "Custom"
	return info, nil
}

// ValidateJavaPath checks if a Java path is valid and compatible.
func (s *SystemService) ValidateJavaPath(path string, requiredMajor int) (*java.JavaInfo, error) {
	if path == "" {
		return nil, NewValidationError("path is required", "path")
	}
	info, err := java.ValidateJavaPath(path, requiredMajor)
	if err != nil {
		return nil, NewIncompatibleError(err.Error())
	}
	return info, nil
}
