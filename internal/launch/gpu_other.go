//go:build !windows

package launch

// ApplyGPUPreference is a no-op outside Windows; Linux uses DRI_PRIME instead.
func ApplyGPUPreference(string, string) error { return nil }
