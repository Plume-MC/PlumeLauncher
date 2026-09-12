//go:build windows

package launch

import (
	"testing"

	"golang.org/x/sys/windows"
)

func TestLaunchHidesConsoleWindow(t *testing.T) {
	cmd, err := Launch(`C:\Program Files\Java\bin\java.exe`, nil, "", nil)
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}
	if cmd.SysProcAttr == nil {
		t.Fatal("SysProcAttr is nil")
	}
	if !cmd.SysProcAttr.HideWindow {
		t.Error("HideWindow = false, want true")
	}
	if cmd.SysProcAttr.CreationFlags&windows.CREATE_NO_WINDOW == 0 {
		t.Errorf("CreationFlags = %#x, want CREATE_NO_WINDOW", cmd.SysProcAttr.CreationFlags)
	}
}
