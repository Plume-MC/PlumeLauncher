//go:build !linux

package launch

// gpuEnvironment has no GPU-selection variables outside Linux.
func gpuEnvironment(string) map[string]string { return nil }
