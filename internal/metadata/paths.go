package metadata

import (
	"strings"
)

// ResolveMavenPath converts a Maven coordinate to a relative path.
// "com.mojang:logging:1.0.0" → "com/mojang/logging/1.0.0/logging-1.0.0.jar"
func ResolveMavenPath(name string) string {
	parts := strings.Split(name, ":")
	if len(parts) != 3 {
		return ""
	}
	group := strings.ReplaceAll(parts[0], ".", "/")
	artifact := parts[1]
	version := parts[2]
	return group + "/" + artifact + "/" + version + "/" + artifact + "-" + version + ".jar"
}

// ResolveNativePath returns the classifier path for a native library.
// Returns ("", false) if the library has no natives for the current OS.
func ResolveNativePath(library Library, sys SystemInfo) (string, bool) {
	classifierKey, ok := GetNativeClassifier(library, sys)
	if !ok {
		return "", false
	}

	// Check classifiers map (newer format)
	if library.Downloads != nil && library.Downloads.Classifiers != nil {
		if dl, ok := library.Downloads.Classifiers[classifierKey]; ok {
			return dl.Path, true
		}
	}

	// Check top-level classifiers (older format)
	if library.Classifiers != nil {
		if dl, ok := library.Classifiers[classifierKey]; ok {
			return dl.Path, true
		}
	}

	return "", false
}

// ResolveLibraryDir returns the base library directory path.
func ResolveLibraryDir() string {
	return "libraries"
}
