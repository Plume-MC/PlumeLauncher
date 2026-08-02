package metadata_test

import (
	"testing"

	"plumelauncher/internal/metadata"
)

func TestCurrentSystemReturnsValidOS(t *testing.T) {
	sys := metadata.CurrentSystem()
	if sys.OS == "" {
		t.Error("OS is empty")
	}
	if sys.Arch == "" {
		t.Error("Arch is empty")
	}
	// Should be one of the known values
	validOS := map[string]bool{"windows": true, "linux": true, "osx": true}
	if !validOS[sys.OS] {
		t.Errorf("OS = %q, not in valid set", sys.OS)
	}
}

func TestShouldDownloadNoRules(t *testing.T) {
	sys := metadata.CurrentSystem()
	if !metadata.ShouldDownload(nil, sys) {
		t.Error("empty rules should allow download")
	}
}

func TestShouldDownloadAllowOnWindows(t *testing.T) {
	sys := metadata.SystemInfo{OS: "windows", Arch: "x64"}
	rules := []metadata.Rule{
		{Action: "allow", OS: &metadata.OSRule{Name: "windows"}},
	}
	if !metadata.ShouldDownload(rules, sys) {
		t.Error("should allow on windows")
	}
}

func TestShouldDownloadDisallowOnOsx(t *testing.T) {
	sys := metadata.SystemInfo{OS: "windows", Arch: "x64"}
	rules := []metadata.Rule{
		{Action: "allow"},
		{Action: "disallow", OS: &metadata.OSRule{Name: "osx"}},
	}
	if !metadata.ShouldDownload(rules, sys) {
		t.Error("should allow on windows (disallow is for osx)")
	}
}

func TestShouldDownloadDisallowOnWindows(t *testing.T) {
	sys := metadata.SystemInfo{OS: "windows", Arch: "x64"}
	rules := []metadata.Rule{
		{Action: "allow"},
		{Action: "disallow", OS: &metadata.OSRule{Name: "osx"}},
	}
	// Disallow osx should not affect windows
	if !metadata.ShouldDownload(rules, sys) {
		t.Error("should allow on windows")
	}
}

func TestShouldDownloadFeaturesRule(t *testing.T) {
	sys := metadata.SystemInfo{
		OS:       "windows",
		Arch:     "x64",
		Features: map[string]bool{"is_demo_user": true},
	}
	rules := []metadata.Rule{
		{Action: "allow", Features: &metadata.FeatureRule{
			IsDemoUser: boolPtr(true),
		}},
	}
	if !metadata.ShouldDownload(rules, sys) {
		t.Error("should allow when demo feature matches")
	}
}

func TestShouldDownloadFeaturesMismatch(t *testing.T) {
	sys := metadata.SystemInfo{
		OS:       "windows",
		Arch:     "x64",
		Features: map[string]bool{"is_demo_user": false},
	}
	rules := []metadata.Rule{
		{Action: "allow", Features: &metadata.FeatureRule{
			IsDemoUser: boolPtr(true),
		}},
	}
	if metadata.ShouldDownload(rules, sys) {
		t.Error("should not allow when demo feature mismatches")
	}
}

func TestGetNativeClassifier(t *testing.T) {
	library := metadata.Library{
		Natives: map[string]string{
			"windows": "natives-windows",
			"linux":   "natives-linux",
			"osx":     "natives-macos",
		},
	}
	sys := metadata.SystemInfo{OS: "windows", Arch: "x64"}
	classifier, ok := metadata.GetNativeClassifier(library, sys)
	if !ok {
		t.Fatal("expected ok")
	}
	if classifier != "natives-windows" {
		t.Errorf("classifier = %q, want %q", classifier, "natives-windows")
	}
}

func TestGetNativeClassifierNoNatives(t *testing.T) {
	library := metadata.Library{Name: "com.mojang:logging:1.0.0"}
	sys := metadata.SystemInfo{OS: "windows", Arch: "x64"}
	_, ok := metadata.GetNativeClassifier(library, sys)
	if ok {
		t.Error("expected false for library without natives")
	}
}

func TestResolveArgumentString(t *testing.T) {
	arg := metadata.Argument{StringValue: "--username"}
	sys := metadata.CurrentSystem()
	result := metadata.ResolveArgument(arg, sys)
	if len(result) != 1 || result[0] != "--username" {
		t.Errorf("result = %v, want [--username]", result)
	}
}

func TestResolveArgumentConditionalPass(t *testing.T) {
	arg := metadata.Argument{
		Conditional: &metadata.ConditionalArgument{
			Rules: []metadata.Rule{
				{Action: "allow", OS: &metadata.OSRule{Name: "windows"}},
			},
			Value: []byte(`"-Xss1M"`),
		},
	}
	sys := metadata.SystemInfo{OS: "windows", Arch: "x64"}
	result := metadata.ResolveArgument(arg, sys)
	if len(result) != 1 || result[0] != "-Xss1M" {
		t.Errorf("result = %v, want [-Xss1M]", result)
	}
}

func TestResolveArgumentConditionalFail(t *testing.T) {
	arg := metadata.Argument{
		Conditional: &metadata.ConditionalArgument{
			Rules: []metadata.Rule{
				{Action: "allow", OS: &metadata.OSRule{Name: "osx"}},
			},
			Value: []byte(`"-XstartOnFirstThread"`),
		},
	}
	sys := metadata.SystemInfo{OS: "windows", Arch: "x64"}
	result := metadata.ResolveArgument(arg, sys)
	if len(result) != 0 {
		t.Errorf("result = %v, want empty", result)
	}
}

func TestResolveArgumentConditionalArray(t *testing.T) {
	arg := metadata.Argument{
		Conditional: &metadata.ConditionalArgument{
			Rules: []metadata.Rule{
				{Action: "allow", Features: &metadata.FeatureRule{
					HasCustomResolution: boolPtr(true),
				}},
			},
			Value: []byte(`["--width", "${resolution_width}", "--height", "${resolution_height}"]`),
		},
	}
	sys := metadata.SystemInfo{
		OS:       "windows",
		Arch:     "x64",
		Features: map[string]bool{"has_custom_resolution": true},
	}
	result := metadata.ResolveArgument(arg, sys)
	if len(result) != 4 {
		t.Errorf("result len = %d, want 4", len(result))
	}
}

func boolPtr(b bool) *bool {
	return &b
}
