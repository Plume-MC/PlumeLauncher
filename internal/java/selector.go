package java

import (
	"fmt"
	"sort"
	"strings"
)

// SelectJava picks the best Java installation for a Minecraft version.
// Returns error if no compatible Java is found.
// Strategy: select the highest patch release of the exact required major.
func SelectJava(installs []JavaInfo, mcVersion string) (*JavaInfo, error) {
	required := RequiredJavaMajor(mcVersion)

	if len(installs) == 0 {
		return nil, fmt.Errorf("no Java found for Minecraft %s (need Java %d+)", mcVersion, required)
	}

	// 1. Exact major match — preferred, pick highest version
	var exactMatches []JavaInfo
	for _, inst := range installs {
		if inst.Major == required {
			exactMatches = append(exactMatches, inst)
		}
	}
	if len(exactMatches) > 0 {
		sort.Slice(exactMatches, func(i, j int) bool {
			return compareJavaVersion(exactMatches[i].Version, exactMatches[j].Version) > 0
		})
		return &exactMatches[0], nil
	}

	return nil, fmt.Errorf("no Java %d found for Minecraft %s (found: %s)", required, mcVersion, formatInstalled(installs))
}

// RequiredJavaMajor returns the required Java major version for a Minecraft version.
func RequiredJavaMajor(mcVersion string) int {
	ver := parseMinecraftVersion(mcVersion)

	// MC 26.x+ → Java 25
	if ver.major >= 26 {
		return 25
	}

	// MC 1.20.5+ → Java 21
	if ver.major == 1 && ver.minor >= 20 && ver.patch >= 5 {
		return 21
	}
	if ver.major > 1 || (ver.major == 1 && ver.minor > 20) {
		return 21
	}

	// MC 1.17 – 1.20.4 → Java 17
	if ver.major == 1 && ver.minor >= 17 {
		return 17
	}

	// MC < 1.17 → Java 8
	return 8
}

type mcVersion struct {
	major, minor, patch int
}

func parseMinecraftVersion(version string) mcVersion {
	// Strip loader suffixes like "-fabric-0.15.0"
	cleaned := version
	if idx := strings.IndexByte(cleaned, '-'); idx != -1 {
		cleaned = cleaned[:idx]
	}

	var v mcVersion
	fmt.Sscanf(cleaned, "%d.%d.%d", &v.major, &v.minor, &v.patch)
	return v
}

func compareJavaVersion(a, b string) int {
	// Extract all numeric groups
	numsA := extractNumbers(a)
	numsB := extractNumbers(b)

	for i := 0; i < len(numsA) && i < len(numsB); i++ {
		if numsA[i] != numsB[i] {
			return numsA[i] - numsB[i]
		}
	}
	return len(numsA) - len(numsB)
}

func extractNumbers(s string) []int {
	var nums []int
	current := 0
	hasDigit := false

	for _, c := range s {
		if c >= '0' && c <= '9' {
			current = current*10 + int(c-'0')
			hasDigit = true
		} else if hasDigit {
			nums = append(nums, current)
			current = 0
			hasDigit = false
		}
	}
	if hasDigit {
		nums = append(nums, current)
	}
	return nums
}

func formatInstalled(installs []JavaInfo) string {
	if len(installs) == 0 {
		return "none"
	}
	var parts []string
	for _, inst := range installs {
		parts = append(parts, fmt.Sprintf("Java %d", inst.Major))
	}
	// Deduplicate
	seen := make(map[string]bool)
	var unique []string
	for _, p := range parts {
		if !seen[p] {
			seen[p] = true
			unique = append(unique, p)
		}
	}
	return strings.Join(unique, ", ")
}
