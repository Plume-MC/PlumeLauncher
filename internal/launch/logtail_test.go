package launch

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// fastTail shrinks the poll interval for one test.
func fastTail(t *testing.T) {
	t.Helper()
	old := logPollInterval
	logPollInterval = 5 * time.Millisecond
	t.Cleanup(func() { logPollInterval = old })
}

// collector records lines delivered by TailLog.
type collector struct {
	lines chan string
}

func newCollector() *collector { return &collector{lines: make(chan string, 32)} }

func (c *collector) collect(line string) {
	select {
	case c.lines <- line:
	default:
	}
}

// waitLine returns the next line or fails the test after a short budget.
func (c *collector) waitLine(t *testing.T) string {
	t.Helper()
	select {
	case line := <-c.lines:
		return line
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for a log line")
		return ""
	}
}

// assertNoLine fails if anything arrives within the grace window.
func (c *collector) assertNoLine(t *testing.T) {
	t.Helper()
	select {
	case line := <-c.lines:
		t.Fatalf("unexpected line %q", line)
	case <-time.After(60 * time.Millisecond):
	}
}

// appendLine writes one line to an existing log file.
func appendLine(t *testing.T, path, line string) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("append: %v", err)
	}
	defer f.Close()
	if _, err := f.WriteString(line); err != nil {
		t.Fatalf("append: %v", err)
	}
}

func TestTailLogSkipsPreexistingContent(t *testing.T) {
	fastTail(t)
	path := filepath.Join(t.TempDir(), "latest.log")
	if err := os.WriteFile(path, []byte("old line\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	c := newCollector()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go TailLog(ctx, path, c.collect)

	// The tailer records the initial end, so "old line" must never surface.
	c.assertNoLine(t)

	appendLine(t, path, "new line\n")
	if line := c.waitLine(t); line != "new line" {
		t.Fatalf("line = %q, want %q", line, "new line")
	}
}

func TestTailLogDeliversAppendedLines(t *testing.T) {
	fastTail(t)
	path := filepath.Join(t.TempDir(), "latest.log")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	file.Close()
	defer os.Remove(path)

	c := newCollector()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go TailLog(ctx, path, c.collect)

	// Let the tailer record the initial end-of-file.
	time.Sleep(20 * time.Millisecond)

	appendLine(t, path, "Connecting to play.example.com, 25565\n")
	if line := c.waitLine(t); line != "Connecting to play.example.com, 25565" {
		t.Fatalf("line = %q", line)
	}

	appendLine(t, path, "Disconnecting\n")
	if line := c.waitLine(t); line != "Disconnecting" {
		t.Fatalf("line = %q", line)
	}
}

func TestTailLogRestartsAfterReplacement(t *testing.T) {
	fastTail(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "latest.log")
	if err := os.WriteFile(path, []byte("first line one\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	c := newCollector()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go TailLog(ctx, path, c.collect)

	c.assertNoLine(t)

	// Minecraft rotates by replacing the file, not truncating in place.
	// A replacement is detected by identity, so the new content is read
	// from the start even though it is longer than the old baseline.
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("second line two\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if line := c.waitLine(t); line != "second line two" {
		t.Fatalf("line = %q", line)
	}
}

func TestTailLogCarriesPartialLineAcrossPolls(t *testing.T) {
	fastTail(t)
	path := filepath.Join(t.TempDir(), "latest.log")
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}

	c := newCollector()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go TailLog(ctx, path, c.collect)

	// The tailer must record the initial end before the partial arrives.
	time.Sleep(20 * time.Millisecond)
	appendLine(t, path, "Connecting to play.example")
	c.assertNoLine(t)
	appendLine(t, path, ".com, 25565\n")
	if line := c.waitLine(t); line != "Connecting to play.example.com, 25565" {
		t.Fatalf("line = %q", line)
	}
}

func TestTailLogToleratesMissingFile(t *testing.T) {
	fastTail(t)
	path := filepath.Join(t.TempDir(), "not-created-yet", "latest.log")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		TailLog(ctx, path, func(string) {})
		close(done)
	}()

	time.Sleep(30 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("TailLog did not stop after cancel")
	}
}

func TestTailLogEmitsNothingBeforeFileAppears(t *testing.T) {
	fastTail(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "latest.log")

	c := newCollector()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go TailLog(ctx, path, c.collect)

	time.Sleep(20 * time.Millisecond)
	if err := os.WriteFile(path, []byte("written later\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if line := c.waitLine(t); line != "written later" {
		t.Fatalf("line = %q", line)
	}
}
