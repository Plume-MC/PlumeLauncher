//go:build windows

package presence

import (
	"net"

	"github.com/Microsoft/go-winio"
)

// dialPath connects to a Discord IPC named pipe. net.Dial has no "pipe"
// network, so the Windows path needs the named pipe API.
func dialPath(path string) (net.Conn, error) {
	timeout := dialTimeout
	return winio.DialPipe(path, &timeout)
}
