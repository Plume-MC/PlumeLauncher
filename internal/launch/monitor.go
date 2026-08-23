package launch

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"sync"
)

// ProcessExitError reports a non-zero process exit code.
type ProcessExitError struct {
	Code int
}

func (e *ProcessExitError) Error() string {
	return fmt.Sprintf("process exited with code %d", e.Code)
}

// LogCallback is called for each line of game output.
type LogCallback func(line string, isStderr bool)

// Monitor starts a process and streams its stdout/stderr.
// Returns when the process exits.
func Monitor(cmd *exec.Cmd, onLog LogCallback) error {
	return MonitorWithStart(cmd, onLog, nil)
}

// MonitorWithStart starts a process, invokes onStarted after a successful
// start, streams output, and waits for process termination.
func MonitorWithStart(cmd *exec.Cmd, onLog LogCallback, onStarted func()) error {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}
	if onStarted != nil {
		onStarted()
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		streamLog(stdout, false, onLog)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		streamLog(stderr, true, onLog)
	}()

	wg.Wait()

	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return &ProcessExitError{Code: exitErr.ExitCode()}
		}
		return fmt.Errorf("process exited with error: %w", err)
	}

	return nil
}

func streamLog(r io.Reader, isStderr bool, onLog LogCallback) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if onLog != nil {
			onLog(line, isStderr)
		}
	}
}
