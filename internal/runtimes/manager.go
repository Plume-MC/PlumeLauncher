package runtimes

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// RuntimeManager manages installed managed JDK runtimes.
type RuntimeManager struct {
	dataRoot string
}

// NewRuntimeManager creates a manager for the given data root.
func NewRuntimeManager(dataRoot string) *RuntimeManager {
	return &RuntimeManager{dataRoot: dataRoot}
}

// ListManaged returns all installed managed JDK runtimes.
func (m *RuntimeManager) ListManaged() []ManagedRuntime {
	base := filepath.Join(m.dataRoot, "runtimes", "java")
	entries, err := os.ReadDir(base)
	if err != nil {
		return nil
	}

	var result []ManagedRuntime
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		manifestPath := filepath.Join(base, entry.Name(), "manifest.json")
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			continue
		}

		var man manifest
		if err := json.Unmarshal(data, &man); err != nil {
			continue
		}

		javaPath := findJavaBinary(filepath.Join(base, entry.Name()))
		result = append(result, ManagedRuntime{
			Major:     man.Major,
			Version:   man.Version,
			Path:      javaPath,
			Installed: javaPath != "",
		})
	}

	return result
}

// GetPath returns the java executable path for the given major version, or error if not installed.
func (m *RuntimeManager) GetPath(major int) (string, error) {
	runtimeDir := filepath.Join(m.dataRoot, "runtimes", "java", fmt.Sprintf("%d", major))

	manifestPath := filepath.Join(runtimeDir, "manifest.json")
	if _, err := os.Stat(manifestPath); err != nil {
		return "", fmt.Errorf("Java %d not installed", major)
	}

	javaPath := findJavaBinary(runtimeDir)
	if javaPath == "" {
		return "", fmt.Errorf("Java %d binary not found", major)
	}

	return javaPath, nil
}

// IsInstalled reports whether a managed JDK is installed for the given major version.
func (m *RuntimeManager) IsInstalled(major int) bool {
	manifestPath := filepath.Join(m.dataRoot, "runtimes", "java", fmt.Sprintf("%d", major), "manifest.json")
	_, err := os.Stat(manifestPath)
	return err == nil
}

// Delete removes a managed JDK runtime for the given major version.
func (m *RuntimeManager) Delete(major int) error {
	runtimeDir := filepath.Join(m.dataRoot, "runtimes", "java", fmt.Sprintf("%d", major))
	if _, err := os.Stat(runtimeDir); err != nil {
		return fmt.Errorf("Java %d not installed", major)
	}
	return os.RemoveAll(runtimeDir)
}

// findJavaBinary locates the java executable inside a JDK directory.
func findJavaBinary(jdkDir string) string {
	var javaName string
	if runtime.GOOS == "windows" {
		javaName = "java.exe"
	} else {
		javaName = "java"
	}

	binDir := filepath.Join(jdkDir, "bin", javaName)
	if _, err := os.Stat(binDir); err == nil {
		return binDir
	}

	// Try without top-level dir stripping (in case extraction didn't strip)
	entries, err := os.ReadDir(filepath.Join(jdkDir, "bin"))
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if e.Name() == javaName || e.Name() == "java" {
			return filepath.Join(jdkDir, "bin", e.Name())
		}
	}

	return ""
}

func writeManifest(runtimeDir string, m manifest) error {
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	manifestPath := filepath.Join(runtimeDir, "manifest.json")
	return os.WriteFile(manifestPath, data, 0o644)
}

// Ensure at least one exported symbol is used to avoid "imported and not used" errors.
var _ = time.Now
