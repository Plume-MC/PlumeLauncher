//go:build !windows

package java

import "os/exec"

// scanWindowsCandidates is a no-op on non-Windows platforms.
func scanWindowsCandidates() []string {
	return nil
}

// hideWindowOnWindows is a no-op on non-Windows platforms.
func hideWindowOnWindows(cmd *exec.Cmd) {
	// no-op
}
