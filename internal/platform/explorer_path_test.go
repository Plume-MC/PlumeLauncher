package platform

import (
	"path/filepath"
	"testing"
)

func TestExplorerExecutableUsesSystemRoot(t *testing.T) {
	got := explorerExecutable(func(string) string { return `D:\Win` })
	want := filepath.Join(`D:\Win`, "explorer.exe")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestExplorerExecutableFallsBackWithoutSystemRoot(t *testing.T) {
	got := explorerExecutable(func(string) string { return "" })
	want := filepath.Join(`C:\Windows`, "explorer.exe")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
