//go:build windows

package java

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/windows/registry"
)

// scanWindowsCandidates returns candidate java binary paths from Windows sources.
func scanWindowsCandidates() []string {
	var candidates []string

	// JAVA_HOME
	if javaHome := os.Getenv("JAVA_HOME"); javaHome != "" {
		candidates = append(candidates, filepath.Join(javaHome, "bin", "java.exe"))
	}

	// PATH directories
	for _, dir := range strings.Split(os.Getenv("PATH"), ";") {
		candidates = append(candidates, filepath.Join(dir, "java.exe"))
	}

	// Common Windows installation directories
	programFiles := []string{
		os.Getenv("ProgramFiles"),
		os.Getenv("ProgramFiles(x86)"),
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs"),
	}
	for _, pf := range programFiles {
		if pf == "" {
			continue
		}
		subdirs := []string{"Java", "Eclipse Foundation", "Microsoft", "Amazon Corretto", "BellSoft", "Zulu"}
		for _, sub := range subdirs {
			root := filepath.Join(pf, sub)
			addJavaFromRoot(root, &candidates)
		}
	}

	// Windows Registry
	candidates = append(candidates, scanRegistryJava()...)

	return candidates
}

// scanRegistryJava reads Java installations from Windows Registry.
func scanRegistryJava() []string {
	var paths []string

	// Try Java Runtime Environment first
	jrePath := scanRegistryKey(`SOFTWARE\JavaSoft\Java Runtime Environment`)
	paths = append(paths, jrePath...)

	// Then Java Development Kit
	jdkPath := scanRegistryKey(`SOFTWARE\JavaSoft\Java Development Kit`)
	paths = append(paths, jdkPath...)

	return paths
}

func scanRegistryKey(keyPath string) []string {
	var paths []string

	key, err := registry.OpenKey(registry.LOCAL_MACHINE, keyPath, registry.READ)
	if err != nil {
		return paths
	}
	defer key.Close()

	subkeys, err := key.ReadSubKeyNames(-1)
	if err != nil {
		return paths
	}

	for _, subkey := range subkeys {
		sub, err := registry.OpenKey(key, subkey, registry.READ)
		if err != nil {
			continue
		}
		javaHome, _, err := sub.GetStringValue("JavaHome")
		sub.Close()
		if err != nil || javaHome == "" {
			continue
		}
		javaBin := filepath.Join(javaHome, "bin", "java.exe")
		if _, err := os.Stat(javaBin); err == nil {
			paths = append(paths, javaBin)
		}
	}

	return paths
}

// addJavaFromRoot walks a root directory looking for Java installations.
func addJavaFromRoot(root string, paths *[]string) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		subdir := filepath.Join(root, entry.Name())
		// Check <subdir>/bin/java.exe
		javaBin := filepath.Join(subdir, "bin", "java.exe")
		if _, err := os.Stat(javaBin); err == nil {
			*paths = append(*paths, javaBin)
			continue
		}
		// Check <subdir>/jre/bin/java.exe (legacy)
		jreBin := filepath.Join(subdir, "jre", "bin", "java.exe")
		if _, err := os.Stat(jreBin); err == nil {
			*paths = append(*paths, jreBin)
		}
	}
}

// hideWindowOnWindows sets the window to be hidden for cmd execution.
func hideWindowOnWindows(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
}
