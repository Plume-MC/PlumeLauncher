package services

import (
	"context"
	"os"
	"path/filepath"

	"plumelauncher/internal/bootstrap"
	"plumelauncher/internal/instances"
	"plumelauncher/internal/java"
	"plumelauncher/internal/launch"
	"plumelauncher/internal/platform"
	"plumelauncher/internal/runtimes"
)

// SystemService manages launcher settings and system operations.
type SystemService struct {
	DataRoot string // canonical launcher data root
	Defaults instances.LauncherDefaults
	JavaDL   *runtimes.JavaDownloader
	RuntimeM *runtimes.RuntimeManager
	Emit     func(string, any) // event emitter
}

// GetSettings returns the current launcher settings.
func (s *SystemService) GetSettings() instances.LauncherDefaults {
	return s.Defaults
}

// UpdateSettings updates the launcher settings.
func (s *SystemService) UpdateSettings(settings instances.LauncherDefaults) error {
	// Validate RAM bounds
	if settings.DefaultMinRamMB < 0 || settings.DefaultMinRamMB > 65536 {
		return NewValidationError("Default minimum RAM must be between 0 and 65536 MB.", "defaultMinRamMB")
	}
	if settings.DefaultMaxRamMB < 0 || settings.DefaultMaxRamMB > 65536 {
		return NewValidationError("Default maximum RAM must be between 0 and 65536 MB.", "defaultMaxRamMB")
	}
	if settings.DefaultMinRamMB > settings.DefaultMaxRamMB {
		return NewValidationError("Default minimum RAM must not exceed maximum RAM.", "defaultMinRamMB", "defaultMaxRamMB")
	}
	if !validGPUPreference(settings.GPUPreference) {
		return NewValidationError("Pick a valid graphics option.", "gpuPreference")
	}
	if !validWrapper(settings.WrapperCommand) {
		return NewValidationError("The wrapper command is invalid. Check Settings for the correct format.", "wrapperCommand")
	}
	if settings.DefaultJavaPath != "" {
		if _, err := java.ValidateJavaPath(settings.DefaultJavaPath, 0); err != nil {
			return NewValidationError("The default Java path does not work. Pick another Java.", "defaultJavaPath")
		}
	}

	if err := instances.SaveConfig(s.DataRoot, settings); err != nil {
		return NewInternalError("Unable to save launcher settings.")
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
		return NewValidationError("Enter a data folder path.", "dataRoot")
	}
	if err := bootstrap.SetDataRoot(path); err != nil {
		return NewValidationError("That data folder path is invalid.", "dataRoot")
	}
	return nil
}

// OpenLogFolder opens the canonical data root's log directory.
func (s *SystemService) OpenLogFolder() error {
	logDir := filepath.Join(s.DataRoot, "logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return NewInternalError("Unable to open the logs folder.")
	}
	if err := platform.OpenFileManager(logDir); err != nil {
		return NewInternalError("Unable to open the logs folder.")
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
		return NewInternalError("Unable to open the " + label + " folder.")
	}
	if err := platform.OpenFileManager(path); err != nil {
		return NewInternalError("Unable to open the " + label + " folder.")
	}
	return nil
}

// RequiredJavaMajor returns the minimum Java major version for a Minecraft
// version, e.g. 21 for "1.20.1". Used by the UI for compatibility guidance.
func (s *SystemService) RequiredJavaMajor(mcVersion string) int {
	return java.RequiredJavaMajor(mcVersion)
}

// ScanJava detects installed Java installations.
func (s *SystemService) ScanJava() ([]java.JavaInfo, error) {
	installed, err := java.RescanJavaInstallations()
	if err != nil {
		return nil, NewInternalError("Unable to scan for Java installations.")
	}
	return installed, nil
}

// JavaRuntimes returns all verified runtimes, marking the PATH runtime and custom entries.
func (s *SystemService) JavaRuntimes() ([]java.JavaInfo, error) {
	installed, err := java.ScanJavaInstallations()
	if err != nil {
		return nil, NewInternalError("Unable to scan for Java installations.")
	}
	for i := range installed {
		installed[i].Source = "Detected"
	}
	if system, err := java.SystemDefault(); err == nil && system != nil {
		installed = append([]java.JavaInfo{*system}, installed...)
	}
	if s.RuntimeM != nil {
		for _, managed := range s.RuntimeM.ListManaged() {
			if !managed.Installed || managed.Path == "" {
				continue
			}
			if info, err := java.ValidateJavaPath(managed.Path, 0); err == nil {
				info.Source = "Managed"
				installed = append(installed, *info)
			}
		}
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
		return nil, NewValidationError("That Java path does not work. Check the path and try again.", "javaPath")
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
		return nil, NewInternalError("Unable to save the Java path.")
	}
	s.Defaults = next
	info.Source = "Custom"
	return info, nil
}

// ValidateJavaPath checks if a Java path is valid and compatible.
func (s *SystemService) ValidateJavaPath(path string, requiredMajor int) (*java.JavaInfo, error) {
	if path == "" {
		return nil, NewValidationError("Enter a Java path.", "path")
	}
	info, err := java.ValidateJavaPath(path, requiredMajor)
	if err != nil {
		return nil, NewIncompatibleError("That Java does not work for this Minecraft version. Pick another Java.")
	}
	return info, nil
}

// DownloadJava starts downloading a managed JDK for the given major version.
func (s *SystemService) DownloadJava(major int) error {
	if s.JavaDL == nil {
		return NewRuntimeError("Java downloads are unavailable right now. Restart the launcher and try again.")
	}
	go func() {
		err := s.JavaDL.Download(context.Background(), major, s.Emit)
		if err != nil && s.Emit != nil {
			s.Emit(EventLogLine, LogLineEvent{Level: "error", Message: "Java download failed: " + err.Error()})
		}
	}()
	return nil
}

// CancelJavaDownload cancels an active JDK download.
func (s *SystemService) CancelJavaDownload(major int) error {
	if s.JavaDL == nil {
		return NewRuntimeError("Java downloads are unavailable right now. Restart the launcher and try again.")
	}
	return s.JavaDL.Cancel(major)
}

// ListManagedRuntimes returns all installed managed JDK runtimes.
func (s *SystemService) ListManagedRuntimes() []runtimes.ManagedRuntime {
	if s.RuntimeM == nil {
		return []runtimes.ManagedRuntime{}
	}
	return s.RuntimeM.ListManaged()
}

// DeleteManagedRuntime removes a managed JDK runtime.
func (s *SystemService) DeleteManagedRuntime(major int) error {
	if s.RuntimeM == nil {
		return NewRuntimeError("Java downloads are unavailable right now. Restart the launcher and try again.")
	}
	return s.RuntimeM.Delete(major)
}

// IsJavaAvailable reports whether a Java runtime is available for the given major version.
func (s *SystemService) IsJavaAvailable(major int) bool {
	// Check managed runtimes first
	if s.RuntimeM != nil && s.RuntimeM.IsInstalled(major) {
		return true
	}
	// Check system installations
	installed, err := java.ScanJavaInstallations()
	if err != nil {
		return false
	}
	for _, info := range installed {
		if info.Major == major {
			return true
		}
	}
	return false
}
