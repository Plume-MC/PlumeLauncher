//go:build !windows

package presence

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"
)

// fakeDiscord is a minimal IPC server that records received frames and
// replies with a READY/error payload per its behavior knobs.
type fakeDiscord struct {
	listener net.Listener
	mu       chan struct{} // closed when server loop exits
	received chan frame
	replyEvt string // "ready" or "error"
}

func startFakeDiscord(t *testing.T, replyEvt string) *fakeDiscord {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "discord-ipc-0")
	listener, err := net.Listen("unix", path)
	if err != nil {
		t.Skipf("unix sockets unavailable: %v", err)
	}
	t.Setenv("XDG_RUNTIME_DIR", dir)
	f := &fakeDiscord{listener: listener, received: make(chan frame, 16), replyEvt: replyEvt, mu: make(chan struct{})}
	go f.serve()
	t.Cleanup(func() {
		_ = listener.Close()
		<-f.mu
	})
	return f
}

func (f *fakeDiscord) serve() {
	defer close(f.mu)
	conn, err := f.listener.Accept()
	if err != nil {
		return
	}
	defer conn.Close()
	for {
		fr, err := readFrame(conn)
		if err != nil {
			return
		}
		f.received <- fr
		evt := f.replyEvt
		payload, _ := json.Marshal(responseData{Evt: evt})
		var header [8]byte
		binary.LittleEndian.PutUint32(header[0:4], uint32(opFrame))
		binary.LittleEndian.PutUint32(header[4:8], uint32(len(payload)))
		conn.Write(header[:])
		conn.Write(payload)
	}
}

func (f *fakeDiscord) expectFrame(t *testing.T) frame {
	t.Helper()
	select {
	case fr := <-f.received:
		return fr
	default:
		t.Fatal("no frame received")
		return frame{}
	}
}

func TestEncodeDecodeFrameRoundTrip(t *testing.T) {
	payload := []byte(`{"v":"1"}`)
	encoded := encodeFrame(opHandshake, payload)
	op := binary.LittleEndian.Uint32(encoded[0:4])
	length := binary.LittleEndian.Uint32(encoded[4:8])
	if op != opHandshake || int(length) != len(payload) {
		t.Fatalf("header mismatch: op=%d len=%d", op, length)
	}
	decoded, err := readFrame(bufio.NewReader(io.NopCloser(newByteReader(encoded))))
	if err != nil {
		t.Fatalf("readFrame: %v", err)
	}
	if decoded.Opcode != opHandshake || string(decoded.Payload) != string(payload) {
		t.Fatalf("payload mismatch: %q", decoded.Payload)
	}
}

func TestReadFrameRejectsOversize(t *testing.T) {
	bad := encodeFrame(opFrame, []byte(`{}`))
	binary.LittleEndian.PutUint32(bad[4:8], maxPayloadSize+1)
	if _, err := readFrame(bufio.NewReader(io.NopCloser(newByteReader(bad)))); err == nil {
		t.Fatal("expected error for oversize payload")
	}
}

// byteReader is a tiny io.Reader over a byte slice.
type byteReader struct{ data []byte }

func newByteReader(b []byte) *byteReader { return &byteReader{data: b} }

func (r *byteReader) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, io.EOF
	}
	n := copy(p, r.data)
	r.data = r.data[n:]
	return n, nil
}

func TestClientHandshakeAndSetActivity(t *testing.T) {
	fake := startFakeDiscord(t, "ready")
	client := NewClient("test-app-id")
	defer client.Close()
	client.Enable()

	if err := client.SetActivity(&Activity{PID: 1234, Details: "Test Instance"}); err != nil {
		t.Fatalf("SetActivity: %v", err)
	}

	hs := fake.expectFrame(t)
	if hs.Opcode != opHandshake {
		t.Fatalf("first frame opcode = %d, want handshake", hs.Opcode)
	}
	var hsPayload handshakePayload
	if err := json.Unmarshal(hs.Payload, &hsPayload); err != nil {
		t.Fatalf("handshake payload: %v", err)
	}
	if hsPayload.ClientID != "test-app-id" {
		t.Errorf("client_id = %q, want test-app-id", hsPayload.ClientID)
	}

	set := fake.expectFrame(t)
	if set.Opcode != opFrame {
		t.Fatalf("second frame opcode = %d, want frame", set.Opcode)
	}
	var env envelope
	if err := json.Unmarshal(set.Payload, &env); err != nil {
		t.Fatalf("envelope: %v", err)
	}
	if env.Cmd != "SET_ACTIVITY" {
		t.Errorf("cmd = %q, want SET_ACTIVITY", env.Cmd)
	}
	if client.Status() != StatusConnected {
		t.Errorf("status = %q, want connected", client.Status())
	}
}

func TestClientRejectsDiscordError(t *testing.T) {
	_ = startFakeDiscord(t, "error")
	client := NewClient("test-app-id")
	defer client.Close()
	client.Enable()

	if err := client.SetActivity(&Activity{PID: 1}); err == nil {
		t.Fatal("expected error when discord replies with evt=error")
	}
	if got := client.Status(); got != StatusError {
		t.Errorf("status = %q, want error", got)
	}
}

func TestClientDisabledNoOp(t *testing.T) {
	client := NewClient("test-app-id")
	defer client.Close()
	if err := client.SetActivity(&Activity{PID: 1}); err != ErrDisabled {
		t.Fatalf("err = %v, want ErrDisabled", err)
	}
	if got := client.Status(); got != StatusDisabled {
		t.Errorf("status = %q, want disabled", got)
	}
}

func TestClientUnavailableWithoutDiscord(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", filepath.Join(t.TempDir(), "empty"))
	t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "empty"))
	client := NewClient("test-app-id")
	defer client.Close()
	client.Enable()

	if err := client.SetActivity(&Activity{PID: 1}); err == nil {
		t.Fatal("expected error when discord socket is absent")
	}
	if got := client.Status(); got != StatusUnavailable {
		t.Errorf("status = %q, want unavailable", got)
	}
}

func TestSocketPathFlatpakSubdir(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "app", "com.discordapp.Discord")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("unix", filepath.Join(nested, "discord-ipc-0"))
	if err != nil {
		t.Skipf("unix sockets unavailable: %v", err)
	}
	defer listener.Close()
	t.Setenv("XDG_RUNTIME_DIR", dir)

	path, err := socketPath(0)
	if err != nil {
		t.Fatalf("socketPath: %v", err)
	}
	if path != filepath.Join(nested, "discord-ipc-0") {
		t.Errorf("path = %q, want flatpak subpath", path)
	}
}
