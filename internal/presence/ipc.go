// Package presence implements the Discord Rich Presence IPC protocol.
// Discord exposes a local socket/pipe (discord-ipc-N) speaking a simple
// binary-framed JSON protocol. No Discord SDK or OAuth is involved; the
// only secret-free payload is a SET_ACTIVITY object.
package presence

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
)

// Opcodes from the Discord IPC protocol.
const (
	opHandshake    = 0
	opFrame        = 1
	opClose        = 2
	opPing         = 3
	opPong         = 4
	maxPayloadSize = 64 * 1024
)

// frame is one IPC message: opcode (u32 LE), length (u32 LE), JSON payload.
type frame struct {
	Opcode  int
	Payload []byte
}

func encodeFrame(opcode int, payload []byte) []byte {
	buf := make([]byte, 8+len(payload))
	binary.LittleEndian.PutUint32(buf[0:4], uint32(opcode))
	binary.LittleEndian.PutUint32(buf[4:8], uint32(len(payload)))
	copy(buf[8:], payload)
	return buf
}

func readFrame(r io.Reader) (frame, error) {
	var header [8]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return frame{}, err
	}
	opcode := int(binary.LittleEndian.Uint32(header[0:4]))
	length := binary.LittleEndian.Uint32(header[4:8])
	if length > maxPayloadSize {
		return frame{}, fmt.Errorf("payload too large: %d", length)
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(r, payload); err != nil {
		return frame{}, err
	}
	return frame{Opcode: opcode, Payload: payload}, nil
}

// handshakePayload is sent first after connecting.
type handshakePayload struct {
	V        string `json:"v"`
	ClientID string `json:"client_id"`
}

// envelope wraps SET_ACTIVITY payloads: {"cmd":..., "args": {...}, "nonce": ...}.
type envelope struct {
	Cmd   string      `json:"cmd"`
	Args  interface{} `json:"args,omitempty"`
	Nonce string      `json:"nonce,omitempty"`
}

// responseData carries the server reply inside a frame payload.
type responseData struct {
	Evt    string          `json:"evt"`
	Data   json.RawMessage `json:"data"`
	Message string         `json:"message"`
}
