//go:build !windows

package java

import (
	"os/exec"
	"path/filepath"
)

// getPlatformCandidates discovers standard Linux JVM installation paths.
func getPlatformCandidates() []string {
	candidates := []string{"/usr/bin/java", "/usr/local/bin/java"}
	if matches, err := filepath.Glob("/usr/lib/jvm/*/bin/java"); err == nil {
		candidates = append(candidates, matches...)
	}
	return candidates
}

// hideWindowOnWindows is a no-op on non-Windows platforms.
func hideWindowOnWindows(cmd *exec.Cmd) {
	// no-op
}
