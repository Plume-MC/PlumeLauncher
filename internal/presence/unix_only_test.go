//go:build !windows

package presence

import "testing"

// Unix-only build tag keeps the fake-server tests out of Windows CI,
// where AF_UNIX fixtures are not portable.
func TestFakeServerRunsOnUnixOnly(t *testing.T) {
	if testing.Short() {
		t.Skip("smoke")
	}
}
