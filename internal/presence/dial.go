package presence

import (
	"net"
	"runtime"
)

// dialPath connects to one IPC endpoint. Unix sockets and Windows named
// pipes share the same framing protocol on the wire.
func dialPath(path string) (net.Conn, error) {
	dialer := net.Dialer{Timeout: dialTimeout}
	network := "unix"
	if runtime.GOOS == "windows" {
		network = "pipe"
	}
	return dialer.Dial(network, path)
}
