package java

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// ScanJavaInstallations discovers all Java installations on the system.
func ScanJavaInstallations() ([]JavaInfo, error) {
	candidates := getCandidates()
	seen := make(map[string]bool)
	var results []JavaInfo

	for _, path := range candidates {
		// Resolve symlinks
		resolved, err := filepath.EvalSymlinks(path)
		if err != nil {
			resolved = path
		}

		if seen[resolved] {
			continue
		}
		seen[resolved] = true

		info, err := CheckJava(path)
		if err != nil || info == nil {
			continue
		}

		// Use resolved path for dedup
		info.Path = resolved
		results = append(results, *info)
	}

	return results, nil
}

// CheckJava runs `java -version` and parses the output.
func CheckJava(path string) (*JavaInfo, error) {
	// Verify file exists
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}

	// Run java -version
	cmd := exec.Command(path, "-version")
	hideWindowOnWindows(cmd)

	// java -version outputs to stderr
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	version := ParseJavaVersion(string(output))
	if version == "" {
		return nil, nil
	}

	major := ParseMajorVersion(version)

	return &JavaInfo{
		Path:    path,
		Version: version,
		Major:   major,
	}, nil
}

func getCandidates() []string {
	var candidates []string

	// JAVA_HOME
	if javaHome := os.Getenv("JAVA_HOME"); javaHome != "" {
		if runtime.GOOS == "windows" {
			candidates = append(candidates, filepath.Join(javaHome, "bin", "java.exe"))
		} else {
			candidates = append(candidates, filepath.Join(javaHome, "bin", "java"))
		}
	}

	// PATH
	if javaPath, err := exec.LookPath("java"); err == nil {
		candidates = append(candidates, javaPath)
	}

	// Platform-specific candidates
	candidates = append(candidates, getPlatformCandidates()...)

	return candidates
}
