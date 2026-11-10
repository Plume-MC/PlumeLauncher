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

// collector records lines delivered by a tailer.
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

// startTail builds the tailer (recording the baseline synchronously) and
// then starts its poll loop.
func startTail(t *testing.T, path string, c *collector) {
	t.Helper()
	tailer := NewTailer(path, c.collect)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go tailer.Run(ctx)
}

func TestTailLogSkipsPreexistingContent(t *testing.T) {
	fastTail(t)
	path := filepath.Join(t.TempDir(), "latest.log")
	if err := os.WriteFile(path, []byte("old line\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	c := newCollector()
	startTail(t, path, c)

	// The baseline was recorded before the goroutine started, so the old
	// line can never surface.
	c.assertNoLine(t)

	appendLine(t, path, "new line\n")
	if line := c.waitLine(t); line != "new line" {
		t.Fatalf("line = %q, want %q", line, "new line")
	}
}

func TestTailLogDeliversAppendedLines(t *testing.T) {
	fastTail(t)
	path := filepath.Join(t.TempDir(), "latest.log")
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}

	c := newCollector()
	startTail(t, path, c)

	appendLine(t, path, "Connecting to play.example.com, 25565\n")
	if line := c.waitLine(t); line != "Connecting to play.example.com, 25565" {
		t.Fatalf("line = %q", line)
	}

	appendLine(t, path, "Disconnecting\n")
	if line := c.waitLine(t); line != "Disconnecting" {
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
	startTail(t, path, c)

	appendLine(t, path, "Connecting to play.example")
	c.assertNoLine(t)
	appendLine(t, path, ".com, 25565\n")
	if line := c.waitLine(t); line != "Connecting to play.example.com, 25565" {
		t.Fatalf("line = %q", line)
	}
}

func TestTailLogRestartsAfterReplacement(t *testing.T) {
	fastTail(t)
	path := filepath.Join(t.TempDir(), "latest.log")
	if err := os.WriteFile(path, []byte("first line one\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	c := newCollector()
	startTail(t, path, c)
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

func TestTailLogEmitsNothingBeforeFileAppears(t *testing.T) {
	fastTail(t)
	path := filepath.Join(t.TempDir(), "latest.log")

	c := newCollector()
	startTail(t, path, c)

	// A log that did not exist at construction is read from the start,
	// because the game creates it fresh for this launch.
	if err := os.WriteFile(path, []byte("written later\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if line := c.waitLine(t); line != "written later" {
		t.Fatalf("line = %q", line)
	}
}

func TestTailLogToleratesMissingFile(t *testing.T) {
	fastTail(t)
	path := filepath.Join(t.TempDir(), "not-created-yet", "latest.log")

	tailer := NewTailer(path, func(string) {})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		tailer.Run(ctx)
		close(done)
	}()

	time.Sleep(30 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("Tailer did not stop after cancel")
	}
}

func TestTailerReportsOnlyLinesAfterConstruction(t *testing.T) {
	fastTail(t)
	path := filepath.Join(t.TempDir(), "latest.log")
	if err := os.WriteFile(path, []byte("before\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	c := newCollector()
	tailer := NewTailer(path, c.collect)

	// Written after construction but before Run starts: still new content.
	appendLine(t, path, "after\n")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go tailer.Run(ctx)

	if line := c.waitLine(t); line != "after" {
		t.Fatalf("line = %q, want %q", line, "after")
	}
}
// TestTailLogDetectsInPlaceRewrite covers the case where the log is rewritten
// without changing size or inode, which is how a fast rotation or a container
// filesystem can look. Size and identity alone cannot detect it, so the tailer
// must compare content at its read position.
func TestTailLogDetectsInPlaceRewrite(t *testing.T) {
	fastTail(t)
	path := filepath.Join(t.TempDir(), "latest.log")
	if err := os.WriteFile(path, []byte("first line\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	c := newCollector()
	startTail(t, path, c)
	c.assertNoLine(t)

	// Same length, same inode, new content.
	if err := os.WriteFile(path, []byte("other line\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if line := c.waitLine(t); line != "other line" {
		t.Fatalf("line = %q", line)
	}
}
