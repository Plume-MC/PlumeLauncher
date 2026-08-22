package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

const (
	defaultMaxBytes = 1 << 20
	defaultMaxFiles = 3
)

var sensitiveText = regexp.MustCompile(`(?i)(token|password|access[_-]?token)([=: ]+)([^\s&,]+)`)

// Logger writes structured JSON lines to a bounded launcher log.
type Logger struct {
	writer  *rotatingWriter
	log     *slog.Logger
	mu      sync.RWMutex
	secrets []string
}

// New creates a logger in dataRoot/logs with bounded rotation and retention.
func New(dataRoot string) (*Logger, error) {
	return NewWithLimits(filepath.Join(dataRoot, "logs"), defaultMaxBytes, defaultMaxFiles)
}

// NewWithLimits keeps rotation tests deterministic without changing production limits.
func NewWithLimits(logDir string, maxBytes int64, maxFiles int) (*Logger, error) {
	writer, err := newRotatingWriter(logDir, maxBytes, maxFiles)
	if err != nil {
		return nil, err
	}
	l := &Logger{writer: writer}
	l.log = slog.New(&redactingHandler{base: slog.NewJSONHandler(writer, &slog.HandlerOptions{}), logger: l})
	return l, nil
}

// AddSecrets registers values that must not appear in logs or emitted messages.
func (l *Logger) AddSecrets(secrets ...string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, secret := range secrets {
		if secret != "" && !contains(l.secrets, secret) {
			l.secrets = append(l.secrets, secret)
		}
	}
}

func (l *Logger) Info(msg string, args ...any)  { l.log.Info(msg, args...) }
func (l *Logger) Warn(msg string, args ...any)  { l.log.Warn(msg, args...) }
func (l *Logger) Error(msg string, args ...any) { l.log.Error(msg, args...) }

// Redact removes registered secrets and common credential values from text.
func (l *Logger) Redact(value string, secrets ...string) string {
	l.mu.RLock()
	all := append([]string(nil), l.secrets...)
	l.mu.RUnlock()
	return Redact(value, append(all, secrets...)...)
}

// Close flushes and closes the active log file.
func (l *Logger) Close() error { return l.writer.Close() }

// Redact removes known secrets and common credential values from text.
func Redact(value string, secrets ...string) string {
	for _, secret := range secrets {
		if secret != "" {
			value = strings.ReplaceAll(value, secret, "[redacted]")
		}
	}
	return sensitiveText.ReplaceAllString(value, `${1}${2}[redacted]`)
}

type redactingHandler struct {
	base   slog.Handler
	logger *Logger
}

func (h *redactingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.base.Enabled(ctx, level)
}

func (h *redactingHandler) Handle(ctx context.Context, record slog.Record) error {
	clean := slog.NewRecord(record.Time, record.Level, h.logger.Redact(record.Message), record.PC)
	record.Attrs(func(attr slog.Attr) bool {
		clean.AddAttrs(sanitizeAttr(attr, h.logger))
		return true
	})
	return h.base.Handle(ctx, clean)
}

func (h *redactingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	clean := make([]slog.Attr, len(attrs))
	for i, attr := range attrs {
		clean[i] = sanitizeAttr(attr, h.logger)
	}
	return &redactingHandler{base: h.base.WithAttrs(clean), logger: h.logger}
}

func (h *redactingHandler) WithGroup(name string) slog.Handler {
	return &redactingHandler{base: h.base.WithGroup(name), logger: h.logger}
}

func sanitizeAttr(attr slog.Attr, logger *Logger) slog.Attr {
	key := strings.ToLower(attr.Key)
	if strings.Contains(key, "password") || strings.Contains(key, "token") || strings.Contains(key, "secret") {
		return slog.String(attr.Key, "[redacted]")
	}
	if attr.Value.Kind() == slog.KindString {
		return slog.String(attr.Key, logger.Redact(attr.Value.String()))
	}
	if attr.Value.Kind() == slog.KindAny {
		if err, ok := attr.Value.Any().(error); ok {
			return slog.String(attr.Key, logger.Redact(err.Error()))
		}
		if values, ok := attr.Value.Any().([]string); ok {
			clean := make([]string, len(values))
			for i, value := range values {
				clean[i] = logger.Redact(value)
			}
			return slog.Any(attr.Key, clean)
		}
	}
	return attr
}

type rotatingWriter struct {
	mu       sync.Mutex
	file     *os.File
	path     string
	maxBytes int64
	maxFiles int
}

func newRotatingWriter(logDir string, maxBytes int64, maxFiles int) (*rotatingWriter, error) {
	if maxBytes <= 0 || maxFiles < 1 {
		return nil, fmt.Errorf("invalid logger limits")
	}
	if err := os.MkdirAll(logDir, 0o700); err != nil {
		return nil, fmt.Errorf("create log directory: %w", err)
	}
	path := filepath.Join(logDir, "launcher.log")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open launcher log: %w", err)
	}
	return &rotatingWriter{file: file, path: path, maxBytes: maxBytes, maxFiles: maxFiles}, nil
}

func (w *rotatingWriter) Write(value []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if info, err := w.file.Stat(); err != nil {
		return 0, err
	} else if info.Size()+int64(len(value)) > w.maxBytes {
		if err := w.rotate(); err != nil {
			return 0, err
		}
	}
	return w.file.Write(value)
}

func (w *rotatingWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	return err
}

func (w *rotatingWriter) rotate() error {
	if err := w.file.Close(); err != nil {
		return err
	}
	for i := w.maxFiles; i >= 1; i-- {
		old := fmt.Sprintf("%s.%d", w.path, i)
		if i == w.maxFiles {
			_ = os.Remove(old)
			continue
		}
		if _, err := os.Stat(old); err == nil {
			if err := os.Rename(old, fmt.Sprintf("%s.%d", w.path, i+1)); err != nil {
				return err
			}
		}
	}
	if err := os.Rename(w.path, w.path+".1"); err != nil && !os.IsNotExist(err) {
		return err
	}
	file, err := os.OpenFile(w.path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	w.file = file
	return nil
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

var _ io.Writer = (*rotatingWriter)(nil)
var _ slog.Handler = (*redactingHandler)(nil)
