package metadata

import (
	"encoding/json"
	"regexp"
	"runtime"
	"strings"
)

// SystemInfo holds the detected host system properties.
type SystemInfo struct {
	OS        string // "windows", "linux", "osx"
	Arch      string // "x64", "x86", "arm64"
	OSVersion string
	Features  map[string]bool
}

// CurrentSystem detects the host OS and architecture, normalized to Mojang conventions.
func CurrentSystem() SystemInfo {
	return SystemInfo{
		OS:   normalizeOS(runtime.GOOS),
		Arch: normalizeArch(runtime.GOARCH),
	}
}

func normalizeOS(goos string) string {
	switch goos {
	case "darwin":
		return "osx"
	case "windows":
		return "windows"
	case "linux":
		return "linux"
	default:
		return goos
	}
}

func normalizeArch(goarch string) string {
	switch goarch {
	case "amd64":
		return "x64"
	case "386":
		return "x86"
	case "arm64":
		return "arm64"
	default:
		return goarch
	}
}

// ShouldDownload evaluates a set of rules against the system.
// No rules = always included. If rules exist: default is false (not included),
// last matching rule wins.
func ShouldDownload(rules []Rule, sys SystemInfo) bool {
	if len(rules) == 0 {
		return true
	}
	allowed := false // default: not included when rules exist
	for _, rule := range rules {
		if matchRule(rule, sys) {
			allowed = rule.Action == "allow"
		}
	}
	return allowed
}

func matchRule(rule Rule, sys SystemInfo) bool {
	// No conditions = catch-all, matches everything
	if rule.OS == nil && rule.Features == nil {
		return true
	}
	if rule.OS != nil && !matchOS(rule.OS, sys) {
		return false
	}
	if rule.Features != nil && !matchFeatures(rule.Features, sys) {
		return false
	}
	return true
}

func matchOS(osRule *OSRule, sys SystemInfo) bool {
	if osRule.Name != "" && osRule.Name != sys.OS {
		return false
	}
	if osRule.Arch != "" && osRule.Arch != sys.Arch {
		return false
	}
	if osRule.Version != "" {
		matched, err := regexp.MatchString(osRule.Version, sys.OSVersion)
		if err != nil || !matched {
			return false
		}
	}
	return true
}

func matchFeatures(featureRule *FeatureRule, sys SystemInfo) bool {
	features := sys.Features
	if features == nil {
		features = map[string]bool{}
	}
	checks := []struct {
		want *bool
		key  string
	}{
		{featureRule.IsDemoUser, "is_demo_user"},
		{featureRule.HasCustomResolution, "has_custom_resolution"},
		{featureRule.HasQuickPlaysSupport, "has_quick_plays_support"},
		{featureRule.IsQuickPlaySingleplayer, "is_quick_play_singleplayer"},
		{featureRule.IsQuickPlayMultiplayer, "is_quick_play_multiplayer"},
		{featureRule.IsQuickPlayRealms, "is_quick_play_realms"},
	}
	for _, check := range checks {
		if check.want == nil {
			continue
		}
		if features[check.key] != *check.want {
			return false
		}
	}
	return true
}

// GetNativeClassifier resolves the native classifier key for a library.
func GetNativeClassifier(library Library, sys SystemInfo) (string, bool) {
	if len(library.Natives) == 0 {
		return "", false
	}
	classifierKey, ok := library.Natives[sys.OS]
	if !ok {
		return "", false
	}
	return strings.ReplaceAll(classifierKey, "${arch}", sys.Arch), true
}

// ResolveArgument returns the argument strings if rules pass.
func ResolveArgument(arg Argument, sys SystemInfo) []string {
	if arg.StringValue != "" {
		return []string{arg.StringValue}
	}
	if arg.Conditional != nil {
		if !ShouldDownload(arg.Conditional.Rules, sys) {
			return nil
		}
		// Parse value as string or string array
		var s string
		if err := json.Unmarshal(arg.Conditional.Value, &s); err == nil {
			return []string{s}
		}
		var arr []string
		if err := json.Unmarshal(arg.Conditional.Value, &arr); err == nil {
			return arr
		}
	}
	return nil
}
