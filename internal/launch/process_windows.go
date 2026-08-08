//go:build windows

package launch

import (
	"os"
	"os/exec"
	"syscall"
)

// Launch starts a Java process with hidden window on Windows.
func Launch(javaPath string, args []string, dir string, env map[string]string) (*exec.Cmd, error) {
	cmd := exec.Command(javaPath, args...)
	cmd.Dir = dir
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}

	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	return cmd, nil
}

// Stop terminates a running process.
func Stop(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	return cmd.Process.Kill()
}
