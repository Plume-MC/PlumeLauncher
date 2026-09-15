package metadata

import "reflect"

// MergeVersions merges a child VersionDetail with its parent (from inheritsFrom).
// Child fields take precedence; missing fields are filled from parent.
func MergeVersions(child, parent *VersionDetail) *VersionDetail {
	if parent == nil {
		return child
	}
	result := *child

	if result.AssetIndex.ID == "" {
		result.AssetIndex = parent.AssetIndex
	}
	if result.Assets == "" {
		result.Assets = parent.Assets
	}
	if result.JavaVersion.Component == "" {
		result.JavaVersion = parent.JavaVersion
	}
	if result.MainClass == "" {
		result.MainClass = parent.MainClass
	}
	if result.Type == "" {
		result.Type = parent.Type
	}
	if result.MinimumLauncherVersion == 0 {
		result.MinimumLauncherVersion = parent.MinimumLauncherVersion
	}
	if result.Downloads.Client == nil {
		result.Downloads.Client = parent.Downloads.Client
	}
	if result.Logging == nil {
		result.Logging = parent.Logging
	}

	// Merge arguments: child first, then parent
	if result.Arguments == nil && parent.Arguments != nil {
		result.Arguments = &Arguments{}
	}
	if parent.Arguments != nil {
		if result.Arguments == nil {
			result.Arguments = &Arguments{}
		}
		result.Arguments.Game = append(result.Arguments.Game, parent.Arguments.Game...)
		result.Arguments.JVM = append(result.Arguments.JVM, parent.Arguments.JVM...)
	}

	// Merge libraries: child first, parent appended (skip exact duplicates).
	// Mojang metadata can repeat a name for its native classifiers.
	for _, lib := range parent.Libraries {
		duplicate := false
		for _, existing := range result.Libraries {
			if reflect.DeepEqual(existing, lib) {
				duplicate = true
				break
			}
		}
		if !duplicate {
			result.Libraries = append(result.Libraries, lib)
		}
	}

	// Clear InheritsFrom after merge
	result.InheritsFrom = ""

	return &result
}
