//go:build !windows

package java

import "os/exec"

// getPlatformCandidates is a no-op on non-Windows platforms.
func getPlatformCandidates() []string {
	return nil
}

// hideWindowOnWindows is a no-op on non-Windows platforms.
func hideWindowOnWindows(cmd *exec.Cmd) {
	// no-op
}
