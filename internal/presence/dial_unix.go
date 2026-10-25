//go:build !windows

package presence

import "net"

// dialPath connects to a Discord IPC unix socket.
func dialPath(path string) (net.Conn, error) {
	return net.DialTimeout("unix", path, dialTimeout)
}
