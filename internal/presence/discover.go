package presence

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// dialTimeout bounds a single connection attempt so a hung Discord
// instance can never stall the launcher.
const dialTimeout = 3 * time.Second

// dial connects to the first available Discord IPC socket.
func dial() (net.Conn, error) {
	lastErr := fmt.Errorf("no discord ipc socket found")
	for i := 0; i < 10; i++ {
		path, err := socketPath(i)
		if err != nil {
			continue
		}
		conn, err := dialPath(path)
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

// socketPath returns the platform path for the Nth Discord IPC endpoint,
// including Flatpak/Snap sandbox subpaths used by packaged Discord builds.
func socketPath(n int) (string, error) {
	name := fmt.Sprintf("discord-ipc-%d", n)
	if runtime.GOOS == "windows" {
		return `\\.\pipe\` + name, nil
	}
	var dirs []string
	if xdg := os.Getenv("XDG_RUNTIME_DIR"); xdg != "" {
		dirs = append(dirs, xdg)
		// Flatpak and Snap sandboxes expose the socket in a nested dir.
		dirs = append(dirs,
			filepath.Join(xdg, "app", "com.discordapp.Discord"),
			filepath.Join(xdg, "snap.discord"),
		)
	}
	if tmp := os.Getenv("TMPDIR"); tmp != "" {
		dirs = append(dirs, tmp)
	}
	dirs = append(dirs, "/tmp")
	for _, dir := range dirs {
		p := filepath.Join(dir, name)
		if info, err := os.Stat(p); err == nil && info.Mode()&os.ModeSocket != 0 {
			return p, nil
		}
	}
	return "", fmt.Errorf("socket %s not found", name)
}
