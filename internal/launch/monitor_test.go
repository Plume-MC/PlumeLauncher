package launch_test

import (
	"errors"
	"os"
	"os/exec"
	"testing"

	"plumelauncher/internal/launch"
)

func TestMonitorHelperProcess(t *testing.T) {
	if os.Getenv("PLUME_MONITOR_HELPER") == "1" {
		os.Exit(7)
	}
}

func TestMonitorReportsExitCode(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=TestMonitorHelperProcess")
	cmd.Env = append(os.Environ(), "PLUME_MONITOR_HELPER=1")
	err := launch.Monitor(cmd, nil)
	var exitErr *launch.ProcessExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("error = %v, want ProcessExitError", err)
	}
	if exitErr.Code != 7 {
		t.Fatalf("exit code = %d, want 7", exitErr.Code)
	}
}
