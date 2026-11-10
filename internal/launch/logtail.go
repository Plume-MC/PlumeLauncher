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

const (
	// maxLogLine caps one buffered line so a corrupt log that never writes a
	// newline cannot grow memory without bound.
	maxLogLine = 8 * 1024
	// tailSampleSize is how many bytes before the read position are kept to
	// verify the file was not rewritten underneath the tailer. A rotation can
	// hand the new file the old inode and a size that only grows, so identity
	// and size are not enough to tell append from replace.
	tailSampleSize = 64
)

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
	// sample holds the last bytes consumed before offset. It is compared on
	// every poll to notice a rewrite that leaves size and inode unchanged.
	sample []byte
}

// NewTailer records the current end of the log. A log that already exists
// holds content from an earlier launch, so it is skipped; a log that does
// not exist yet is read from the start once the game creates it.
func NewTailer(path string, onLine func(string)) *Tailer {
	t := &Tailer{path: path, onLine: onLine}
	info, err := os.Stat(path)
	if err != nil {
		return t
	}
	t.prevInfo = info
	t.offset = info.Size()
	t.seedSample()
	return t
}

// seedSample captures the bytes just before the initial read position so a
// later rewrite can be told apart from an append.
func (t *Tailer) seedSample() {
	count := int64(tailSampleSize)
	if t.offset < count {
		count = t.offset
	}
	if count == 0 {
		return
	}
	file, err := os.Open(t.path)
	if err != nil {
		return
	}
	defer file.Close()
	if _, err := file.Seek(t.offset-count, io.SeekStart); err != nil {
		return
	}
	buf := make([]byte, count)
	if _, err := io.ReadFull(file, buf); err != nil {
		return
	}
	t.sample = buf
}

// Run polls the log until ctx is cancelled. A missing, rotated, or rewritten
// file is waited out or restarted rather than reported: this backs best-effort
// POST-005 presence and must never disturb the launch itself.
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

	if t.replaced(cur) {
		t.offset = 0
		t.partial = t.partial[:0]
		t.sample = t.sample[:0]
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

// replaced reports whether cur is no longer the file the tailer was reading.
// Identity catches a normal rotation, and the content check catches a
// replacement that reused the inode or an in-place rewrite.
func (t *Tailer) replaced(cur os.FileInfo) bool {
	if t.prevInfo == nil || !os.SameFile(t.prevInfo, cur) {
		return true
	}
	if cur.Size() < t.offset {
		return true
	}
	return !t.sampleMatches()
}

// sampleMatches verifies the bytes before the read position are unchanged.
func (t *Tailer) sampleMatches() bool {
	count := int64(len(t.sample))
	if count == 0 || t.offset < count {
		// Nothing to compare against yet.
		return true
	}
	file, err := os.Open(t.path)
	if err != nil {
		return false
	}
	defer file.Close()
	if _, err := file.Seek(t.offset-count, io.SeekStart); err != nil {
		return false
	}
	buf := make([]byte, count)
	if _, err := io.ReadFull(file, buf); err != nil {
		return false
	}
	return bytes.Equal(buf, t.sample)
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
		t.appendSample(data[:n])
		t.partial = append(t.partial, data[:n]...)
		t.partial = emitLines(t.partial, t.onLine)
	}
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return int64(n), t.partial, err
	}
	return int64(n), t.partial, nil
}

// appendSample keeps the last tailSampleSize consumed bytes.
func (t *Tailer) appendSample(data []byte) {
	t.sample = append(t.sample, data...)
	if n := len(t.sample); n > tailSampleSize {
		copy(t.sample, t.sample[n-tailSampleSize:])
		t.sample = t.sample[:tailSampleSize]
	}
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