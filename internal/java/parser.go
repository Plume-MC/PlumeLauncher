package java

import (
	"regexp"
	"runtime"
)

// JavaInfo holds information about a detected Java installation.
type JavaInfo struct {
	Path    string `json:"path"`
	Version string `json:"version"`
	Major   int    `json:"major"`
}

var versionRegex = regexp.MustCompile(`version "([^"]+)"`)

// ParseJavaVersion extracts the version string from `java -version` output.
// Example input: `openjdk version "17.0.8" 2023-07-18`
// Returns: "17.0.8"
func ParseJavaVersion(output string) string {
	match := versionRegex.FindStringSubmatch(output)
	if len(match) < 2 {
		return ""
	}
	return match[1]
}

// ParseMajorVersion extracts the major version number from a version string.
// Handles both legacy "1.8.0_382" (returns 8) and modern "17.0.8" (returns 17).
func ParseMajorVersion(version string) int {
	if version == "" {
		return 0
	}

	// Strip any suffixes like "-ea", "+1", "-LTS"
	cleaned := version
	for i, c := range cleaned {
		if c == '-' || c == '+' {
			cleaned = cleaned[:i]
			break
		}
	}

	// Split on dots and underscores
	parts := splitVersion(cleaned)
	if len(parts) == 0 {
		return 0
	}

	// Legacy format: 1.x.y → major is x
	if len(parts) >= 2 && parts[0] == 1 {
		return parts[1]
	}

	// Modern format: N.x.y → major is N
	return parts[0]
}

func splitVersion(s string) []int {
	var result []int
	current := 0
	hasDigit := false

	for _, c := range s {
		if c >= '0' && c <= '9' {
			current = current*10 + int(c-'0')
			hasDigit = true
		} else if hasDigit {
			result = append(result, current)
			current = 0
			hasDigit = false
		}
	}
	if hasDigit {
		result = append(result, current)
	}

	return result
}

// FormatMajorToJavaBinary returns the Java binary name for the platform.
func FormatMajorToJavaBinary() string {
	if runtime.GOOS == "windows" {
		return "java.exe"
	}
	return "java"
}
