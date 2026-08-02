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

// ResolveAssetPath returns the relative path for an asset object using the hash scheme.
// e.g. hash "0945265e5d7c19a3" → "09/0945265e5d7c19a3"
func ResolveAssetPath(hash string) string {
	if len(hash) < 2 {
		return hash
	}
	return hash[:2] + "/" + hash
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

// ResolveClientPath returns the relative path for the client JAR.
func ResolveClientPath(detail VersionDetail) string {
	if detail.Downloads.Client != nil && detail.Downloads.Client.Path != "" {
		return detail.Downloads.Client.Path
	}
	// Fallback: construct from ID
	return "versions/" + detail.ID + "/" + detail.ID + ".jar"
}

// ResolveServerPath returns the relative path for the server JAR, if present.
func ResolveServerPath(detail VersionDetail) (string, bool) {
	if detail.Downloads.Server != nil && detail.Downloads.Server.Path != "" {
		return detail.Downloads.Server.Path, true
	}
	return "", false
}

// ResolveLibraryDir returns the base library directory path.
func ResolveLibraryDir() string {
	return "libraries"
}

// ResolveAssetDir returns the base asset directory path.
func ResolveAssetDir(detail VersionDetail) string {
	return "assets/objects"
}

// ResolveAssetIndexPath returns the asset index JSON path.
func ResolveAssetIndexPath(detail VersionDetail) string {
	return "assets/indexes/" + detail.AssetIndex.ID + ".json"
}

// ResolveVersionJSONPath returns the version JSON path.
func ResolveVersionJSONPath(detail VersionDetail) string {
	return "versions/" + detail.ID + "/" + detail.ID + ".json"
}
