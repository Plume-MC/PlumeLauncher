package java

import (
	"fmt"
	"os"
)

// ValidateJavaPath checks if a custom Java path is executable and matches the
// required Java major version. A zero requirement only validates the binary.
func ValidateJavaPath(path string, requiredMajor int) (*JavaInfo, error) {
	// Check file exists
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("java path not found: %s", path)
	}

	// Check it's actually a Java installation
	info, err := CheckJava(path)
	if err != nil {
		return nil, fmt.Errorf("failed to run java at %s: %w", path, err)
	}
	if info == nil {
		return nil, fmt.Errorf("no valid java version output from %s", path)
	}

	// Minecraft versions require an exact Java major; newer majors are not a safe fallback.
	if requiredMajor > 0 && info.Major != requiredMajor {
		return nil, fmt.Errorf("java at %s is version %d, but exactly Java %d is required", path, info.Major, requiredMajor)
	}

	return info, nil
}
