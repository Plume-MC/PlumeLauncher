package launch

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"time"
)

// logPollInterval bounds how often the game log is scanned for new lines.
// Polling keeps a file-watcher dependency out of the project. It is a
// variable so tests can shrink it.
var logPollInterval = 500 * time.Millisecond

// maxLogLine caps one buffered line so a corrupt log that never writes a
// newline cannot grow memory without bound.
const maxLogLine = 8 * 1024

// LogPath returns the standard client log path inside a game directory.
func LogPath(gameDir string) string {
	return filepath.Join(gameDir, "logs", "latest.log")
}

// Tailer follows a game log and reports complete lines written after it was
// created. Splitting construction from Run makes the starting position
// deterministic: the baseline is recorded by NewTailer, not by whichever
// goroutine happens to run first.
type Tailer struct {
	path   string
	onLine func(string)

	prevInfo os.FileInfo
	offset   int64
	partial  []byte
}

// NewTailer records the current end of the log. A log that already exists
// holds content from an earlier launch, so it is skipped; a log that does
// not exist yet is read from the start once the game creates it.
func NewTailer(path string, onLine func(string)) *Tailer {
	t := &Tailer{path: path, onLine: onLine}
	if info, err := os.Stat(path); err == nil {
		t.prevInfo = info
		t.offset = info.Size()
	}
	return t
}

// Run polls the log until ctx is cancelled. A missing, rotated, or truncated
// file is waited out or restarted rather than reported: this backs
// best-effort POST-005 presence and must never disturb the launch itself.
//
// Known limit: an in-place truncation that regrows past the previous size
// inside one poll window is not detectable. Minecraft replaces the file
// rather than truncating it, so that case does not arise in practice.
func (t *Tailer) Run(ctx context.Context) {
	ticker := time.NewTicker(logPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		t.poll()
	}
}

// poll reads whatever has been appended since the last call.
func (t *Tailer) poll() {
	cur, err := os.Stat(t.path)
	if err != nil {
		// The log was rotated away or is not written yet. Keep the
		// baseline so a replacement is detected when it appears.
		return
	}

	if t.prevInfo == nil || !os.SameFile(t.prevInfo, cur) {
		// First appearance after launch, or a replacement file.
		t.offset = 0
		t.partial = t.partial[:0]
	} else if cur.Size() < t.offset {
		// Same file shrank underneath us.
		t.offset = 0
		t.partial = t.partial[:0]
	}
	t.prevInfo = cur

	if cur.Size() <= t.offset {
		return
	}
	consumed, tail, err := t.readRange(cur.Size() - t.offset)
	if err != nil {
		return
	}
	t.offset += consumed
	t.partial = tail
}

// readRange reads the newly appended bytes, emits the complete lines they
// contain, and returns how many bytes were consumed plus any unterminated
// tail to carry into the next poll.
func (t *Tailer) readRange(count int64) (int64, []byte, error) {
	file, err := os.Open(t.path)
	if err != nil {
		return 0, t.partial, err
	}
	defer file.Close()

	if _, err := file.Seek(t.offset, io.SeekStart); err != nil {
		return 0, t.partial, err
	}
	data := make([]byte, count)
	n, err := io.ReadFull(file, data)
	if n > 0 {
		t.partial = append(t.partial, data[:n]...)
		t.partial = emitLines(t.partial, t.onLine)
	}
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return int64(n), t.partial, err
	}
	return int64(n), t.partial, nil
}

// TailLog is the convenience wrapper used by the launch monitor.
func TailLog(ctx context.Context, path string, onLine func(string)) {
	NewTailer(path, onLine).Run(ctx)
}

// emitLines splits complete lines out of buf, calls onLine for each, and
// returns the unterminated tail.
func emitLines(buf []byte, onLine func(string)) []byte {
	for {
		i := bytes.IndexByte(buf, '\n')
		if i < 0 {
			if len(buf) > maxLogLine {
				// A line this long is corrupt; drop it rather than
				// keep growing memory.
				return nil
			}
			return buf
		}
		line := string(bytes.TrimSpace(buf[:i]))
		buf = buf[i+1:]
		if line != "" {
			onLine(line)
		}
	}
}