//go:build !windows

package presence

import (
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// listenFakeDiscord starts a fake IPC server on a fresh unix socket in dir.
// Each connection performs the handshake and answers every frame with READY.
func listenFakeDiscord(t *testing.T, dir string) (net.Listener, string) {
	t.Helper()
	path := filepath.Join(dir, "discord-ipc-0")
	_ = os.Remove(path)
	listener, err := net.Listen("unix", path)
	if err != nil {
		t.Skipf("unix sockets unavailable: %v", err)
	}
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				for {
					if _, err := readFrame(conn); err != nil {
						return
					}
					writeReady(conn)
				}
			}()
		}
	}()
	return listener, path
}

// writeReady replies to a frame with an evt=ready payload.
func writeReady(conn net.Conn) {
	payload := []byte(`{"evt":"ready"}`)
	header := encodeFrame(opFrame, payload)
	_, _ = conn.Write(header)
}

// fastRetry shrinks retry tuning on one client for the duration of a test.
func fastRetry(t *testing.T, client *Client) {
	t.Helper()
	client.mu.Lock()
	client.retryBackoff = 10 * time.Millisecond
	client.retryAttempts = 8
	client.mu.Unlock()
}

func TestRetryRecoversWhenDiscordReturns(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", dir)

	client := NewClient("test-id")
	defer client.Close()
	fastRetry(t, client)
	client.Enable()

	// Discord absent: publish fails and schedules a retry.
	if err := client.SetActivity(&Activity{PID: 1, Details: "Test"}); err == nil {
		t.Fatal("expected failure while discord is absent")
	}
	if got := client.Status(); got != StatusUnavailable {
		t.Errorf("status = %q, want unavailable", got)
	}

	// Discord appears mid-backoff; the retry loop should publish and stop.
	listener, _ := listenFakeDiscord(t, dir)
	defer listener.Close()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if client.Status() == StatusConnected {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("status = %q after discord returned, want connected", client.Status())
}

func TestDisableStopsRetry(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", dir)

	client := NewClient("test-id")
	defer client.Close()
	fastRetry(t, client)
	client.Enable()

	if err := client.SetActivity(&Activity{PID: 1}); err == nil {
		t.Fatal("expected failure while discord is absent")
	}
	client.Disable()

	// Even with Discord now available, a disabled client must not reconnect.
	listener, _ := listenFakeDiscord(t, dir)
	defer listener.Close()

	time.Sleep(150 * time.Millisecond)
	if got := client.Status(); got != StatusDisabled {
		t.Errorf("status = %q, want disabled (retry must stop)", got)
	}
}

func TestClearWithoutConnectionDoesNotDial(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", dir)
	t.Setenv("TMPDIR", dir)

	client := NewClient("test-id")
	defer client.Close()
	client.Enable()

	// No game was ever published, so clearing is a no-op and must not dial.
	if err := client.SetActivity(nil); err != nil {
		t.Fatalf("clear = %v, want nil no-op", err)
	}
	if got := client.Status(); got != StatusIdle {
		t.Errorf("status = %q, want idle", got)
	}
}

func TestClearAfterStopLeavesIdle(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", dir)
	listener, _ := listenFakeDiscord(t, dir)
	defer listener.Close()

	client := NewClient("test-id")
	defer client.Close()
	client.Enable()

	if err := client.SetActivity(&Activity{PID: 1}); err != nil {
		t.Fatalf("publish: %v", err)
	}
	if err := client.SetActivity(nil); err != nil {
		t.Fatalf("clear: %v", err)
	}
	if got := client.Status(); got != StatusIdle {
		t.Errorf("status = %q, want idle after clear", got)
	}
}

// TestEnableKeepsConnectedStatus guards the case where saving settings while a
// game is running overwrote "connected" with "idle".
func TestEnableKeepsConnectedStatus(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", dir)
	listener, _ := listenFakeDiscord(t, dir)
	defer listener.Close()

	client := NewClient("test-id")
	defer client.Close()
	client.Enable()

	if err := client.SetActivity(&Activity{PID: 1}); err != nil {
		t.Fatalf("publish: %v", err)
	}
	if got := client.Status(); got != StatusConnected {
		t.Fatalf("status = %q, want connected", got)
	}

	// Settings saved mid-game call Enable again on the live client.
	client.Enable()
	if got := client.Status(); got != StatusConnected {
		t.Errorf("status = %q after re-enable, want connected", got)
	}
}

// TestEnableAfterDisconnectReportsIdle covers the cold path.
func TestEnableAfterDisconnectReportsIdle(t *testing.T) {
	client := NewClient("test-id")
	defer client.Close()

	client.Enable()
	client.Disable()
	client.Enable()
	if got := client.Status(); got != StatusIdle {
		t.Errorf("status = %q, want idle", got)
	}
}
