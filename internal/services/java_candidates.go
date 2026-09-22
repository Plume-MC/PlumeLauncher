package services

import (
	"path/filepath"

	"plumelauncher/internal/instances"
	"plumelauncher/internal/java"
	"plumelauncher/internal/runtimes"
)

// launchJavaCandidates returns verified Java runtimes from the system scan,
// managed downloads, and custom paths: the same universe Settings shows.
// Managed and custom entries that fail validation are skipped.
//
// scanJavaInstallations and validateJavaPath are package vars (not direct
// calls) so tests can stub them and stay hermetic across OSes.
var (
	scanJavaInstallations = java.ScanJavaInstallations
	validateJavaPath      = java.ValidateJavaPath
)

func launchJavaCandidates(dataRoot string, defaults instances.LauncherDefaults) ([]java.JavaInfo, error) {
	installed, err := scanJavaInstallations()
	if err != nil {
		return nil, err
	}
	manager := runtimes.NewRuntimeManager(dataRoot)
	for _, managed := range manager.ListManaged() {
		if !managed.Installed || managed.Path == "" {
			continue
		}
		if info, err := validateJavaPath(managed.Path, 0); err == nil {
			installed = append(installed, *info)
		}
	}
	for _, path := range defaults.CustomJavaPaths {
		if info, err := validateJavaPath(path, 0); err == nil {
			installed = append(installed, *info)
		}
	}
	seen := make(map[string]bool)
	candidates := make([]java.JavaInfo, 0, len(installed))
	for _, info := range installed {
		path, err := filepath.EvalSymlinks(info.Path)
		if err == nil {
			info.Path = path
		}
		if !seen[info.Path] {
			seen[info.Path] = true
			candidates = append(candidates, info)
		}
	}
	return candidates, nil
}
