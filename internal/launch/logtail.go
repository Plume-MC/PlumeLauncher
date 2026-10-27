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

// TailLog follows a game log, calling onLine for every complete line written
// after the game started. A log that does not exist yet is read from the
// start once it appears, because the game creates it fresh per launch. A
// pre-existing log is skipped to its end, since its content predates the
// launch. Replacement (rotation) resets the read position to zero.
//
// A missing, rotated, or truncated file is waited out or restarted rather
// than reported: this backs best-effort POST-005 presence and must never
// disturb the launch itself.
//
// Known limit: an in-place truncation that regrows past the previous size
// inside one poll window is not detectable. Minecraft replaces the file
// rather than truncating it, so that case does not arise in practice.
func TailLog(ctx context.Context, path string, onLine func(string)) {
	var (
		prevInfo os.FileInfo
		offset   int64
		partial  []byte
	)

	// A log that already exists holds content from an earlier launch, so
	// start after it. One that does not exist yet is read from zero.
	if info, err := os.Stat(path); err == nil {
		prevInfo = info
		offset = info.Size()
	}

	ticker := time.NewTicker(logPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		cur, err := os.Stat(path)
		if err != nil {
			// The log was rotated away or is not written yet. Keep the
			// baseline so a replacement is detected when it appears.
			continue
		}

		if prevInfo == nil || !os.SameFile(prevInfo, cur) {
			// First appearance after launch, or a replacement file.
			offset = 0
			partial = partial[:0]
		} else if cur.Size() < offset {
			// Same file shrank underneath us.
			offset = 0
			partial = partial[:0]
		}
		prevInfo = cur

		if cur.Size() <= offset {
			continue
		}
		consumed, tail, err := readRange(path, offset, cur.Size()-offset, onLine, partial)
		if err != nil {
			continue
		}
		offset += consumed
		partial = tail
	}
}

// readRange reads the newly appended bytes, emits the complete lines it
// contains, and returns how many bytes were consumed plus any unterminated
// tail to carry into the next poll.
func readRange(path string, offset, count int64, onLine func(string), partial []byte) (int64, []byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, partial, err
	}
	defer file.Close()

	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return 0, partial, err
	}
	data := make([]byte, count)
	n, err := io.ReadFull(file, data)
	if n > 0 {
		partial = append(partial, data[:n]...)
		partial = emitLines(partial, onLine)
	}
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return int64(n), partial, err
	}
	return int64(n), partial, nil
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
