//go:build !windows

package launch

// ApplyGPUPreference is a no-op outside Windows; Linux uses environment variables instead.
func ApplyGPUPreference(string, string) error { return nil }
