package logging_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"plumelauncher/internal/logging"
)

func TestLoggerWritesStructuredRedactedContext(t *testing.T) {
	dir := t.TempDir()
	logger, err := logging.NewWithLimits(dir, 4096, 3)
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()
	logger.AddSecrets("session-secret")
	logger.Info("launch started", "instanceId", "demo", "operationId", "op-1", "token", "session-secret", "args", []string{"-javaagent:authlib", "token=session-secret"})

	data, err := os.ReadFile(filepath.Join(dir, "launcher.log"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, `"instanceId":"demo"`) || !strings.Contains(text, `"operationId":"op-1"`) {
		t.Fatalf("missing structured context: %s", text)
	}
	if strings.Contains(text, "session-secret") || strings.Contains(text, "password=") {
		t.Fatalf("secret persisted in log: %s", text)
	}
}

func TestLoggerRotatesAndRetainsBoundedFiles(t *testing.T) {
	dir := t.TempDir()
	logger, err := logging.NewWithLimits(dir, 100, 3)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20; i++ {
		logger.Info("event", "index", i, "message", strings.Repeat("x", 30))
	}
	if err := logger.Close(); err != nil {
		t.Fatal(err)
	}
	files, err := filepath.Glob(filepath.Join(dir, "launcher.log*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) > 4 {
		t.Fatalf("retained %d log files, want at most 4", len(files))
	}
}

func TestLoggerAccountsForExistingLogSize(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "launcher.log")
	if err := os.WriteFile(path, []byte("existing log\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	logger, err := logging.NewWithLimits(dir, 1, 3)
	if err != nil {
		t.Fatal(err)
	}
	logger.Info("new log entry")
	if err := logger.Close(); err != nil {
		t.Fatal(err)
	}
	archived, err := os.ReadFile(path + ".1")
	if err != nil {
		t.Fatal(err)
	}
	if string(archived) != "existing log\n" {
		t.Fatalf("existing log was not rotated: %q", archived)
	}
}

func TestRedactMasksCredentialSyntax(t *testing.T) {
	got := logging.Redact("password=hunter2 access_token=abc123 token=xyz")
	if strings.Contains(got, "hunter2") || strings.Contains(got, "abc123") || strings.Contains(got, "xyz") {
		t.Fatalf("credentials were not redacted: %s", got)
	}
}

func TestRedactMasksOAuthCallbackURL(t *testing.T) {
	raw := "redirect https://login.live.com/oauth20_desktop.srf?code=AUTHCODE123&state=ok refresh_token=REFRESH123"
	got := logging.Redact(raw)
	if strings.Contains(got, "AUTHCODE123") || strings.Contains(got, "REFRESH123") {
		t.Fatalf("oauth artifacts were not redacted: %s", got)
	}
	if !strings.Contains(got, "code=[redacted]") {
		t.Fatalf("code param must be visibly masked: %s", got)
	}
}
