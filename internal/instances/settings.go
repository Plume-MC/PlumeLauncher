package instances

// LauncherDefaults holds global launcher settings that instances inherit from.
type LauncherDefaults struct {
	Theme                  string   `json:"theme"`
	CloseAction            string   `json:"closeAction"`
	WindowMode             string   `json:"windowMode"`
	DefaultMinRamMB        int      `json:"defaultMinRamMB"`
	DefaultMaxRamMB        int      `json:"defaultMaxRamMB"`
	DefaultResolutionW     int      `json:"defaultResolutionW"`
	DefaultResolutionH     int      `json:"defaultResolutionH"`
	DefaultJavaPath        string   `json:"defaultJavaPath"`
	CustomJavaPaths        []string `json:"customJavaPaths,omitempty"`
	JavaDefaultInitialized bool     `json:"javaDefaultInitialized"`
	DefaultJvmArgs         string   `json:"defaultJvmArgs"`
	GPUPreference          string   `json:"gpuPreference"`
	WrapperCommand         string   `json:"wrapperCommand"`
}

// DefaultLauncherDefaults returns sensible defaults.
func DefaultLauncherDefaults() LauncherDefaults {
	return LauncherDefaults{
		Theme:              "dark",
		CloseAction:        "keep_open",
		WindowMode:         "Windowed",
		DefaultMinRamMB:    1024,
		DefaultMaxRamMB:    4096,
		DefaultResolutionW: 854,
		DefaultResolutionH: 480,
		GPUPreference:      "auto",
	}
}

// EffectiveSettings returns the instance settings with inheritance from launcher defaults.
func EffectiveSettings(inst Instance, defaults LauncherDefaults) EffectiveSettingsResult {
	result := EffectiveSettingsResult{
		InstanceID: inst.ID,
	}

	// Min RAM
	if inst.Settings.MinRamMB != nil {
		result.MinRamMB = *inst.Settings.MinRamMB
	} else {
		result.MinRamMB = defaults.DefaultMinRamMB
	}

	// Max RAM
	if inst.Settings.MaxRamMB != nil {
		result.MaxRamMB = *inst.Settings.MaxRamMB
	} else {
		result.MaxRamMB = defaults.DefaultMaxRamMB
	}

	// Resolution
	if inst.Settings.ResolutionW != nil {
		result.ResolutionW = *inst.Settings.ResolutionW
	} else {
		result.ResolutionW = defaults.DefaultResolutionW
	}
	if inst.Settings.ResolutionH != nil {
		result.ResolutionH = *inst.Settings.ResolutionH
	} else {
		result.ResolutionH = defaults.DefaultResolutionH
	}

	// Java path
	if inst.Settings.JavaPath != nil && *inst.Settings.JavaPath != "" {
		result.JavaPath = *inst.Settings.JavaPath
	} else {
		result.JavaPath = defaults.DefaultJavaPath
	}

	// JVM args
	if inst.Settings.JVMArgs != nil && *inst.Settings.JVMArgs != "" {
		result.JVMArgs = *inst.Settings.JVMArgs
	} else {
		result.JVMArgs = defaults.DefaultJvmArgs
	}

	// Window mode
	if inst.Settings.WindowMode != nil && *inst.Settings.WindowMode != "" {
		result.WindowMode = *inst.Settings.WindowMode
	} else {
		result.WindowMode = defaults.WindowMode
	}

	// GPU preference
	if inst.Settings.GPUPreference != nil && *inst.Settings.GPUPreference != "" {
		result.GPUPreference = *inst.Settings.GPUPreference
	} else {
		result.GPUPreference = defaults.GPUPreference
	}

	// Wrapper command
	if inst.Settings.WrapperCommand != nil && *inst.Settings.WrapperCommand != "" {
		result.WrapperCommand = *inst.Settings.WrapperCommand
	} else {
		result.WrapperCommand = defaults.WrapperCommand
	}

	return result
}

// EffectiveSettingsResult is the resolved settings after inheritance.
type EffectiveSettingsResult struct {
	InstanceID     string
	MinRamMB       int
	MaxRamMB       int
	ResolutionW    int
	ResolutionH    int
	JavaPath       string
	JVMArgs        string
	WindowMode     string
	GPUPreference  string
	WrapperCommand string
}

// StringPtr returns a pointer to the string value.
func StringPtr(s string) *string {
	return &s
}

// IntPtr returns a pointer to the int value.
func IntPtr(i int) *int {
	return &i
}
