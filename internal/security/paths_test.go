package security

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestResolveUnderRootRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	for _, value := range []string{"../outside", "/tmp/outside", `..\outside`, "a/../outside"} {
		if _, err := ResolveUnderRoot(root, value); err == nil {
			t.Errorf("ResolveUnderRoot(%q) accepted traversal", value)
		}
	}
}

func TestResolveUnderRootRejectsSymlinkEscape(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires elevated privileges on some Windows hosts")
	}
	root := t.TempDir()
	outside := t.TempDir()
	link := filepath.Join(root, "link")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := ResolveUnderRoot(root, "link/file.txt"); err == nil {
		t.Fatal("ResolveUnderRoot accepted symlink escape")
	}
}

func TestResolveUnderRootAllowsContainedPath(t *testing.T) {
	root := t.TempDir()
	got, err := ResolveUnderRoot(root, "instances/one/instance.json")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(root, "instances", "one", "instance.json"); got != want {
		t.Fatalf("path = %q, want %q", got, want)
	}
}

func TestValidateArtifactURL(t *testing.T) {
	for _, value := range []string{
		"https://piston-data.mojang.com/v1/objects/hash/client.jar",
		"https://maven.fabricmc.net/example.jar",
	} {
		if err := ValidateArtifactURL(value); err != nil {
			t.Errorf("ValidateArtifactURL(%q): %v", value, err)
		}
	}
	for _, value := range []string{
		"http://piston-data.mojang.com/client.jar",
		"https://evil.example/client.jar",
		"https://user:pass@libraries.minecraft.net/client.jar",
	} {
		if err := ValidateArtifactURL(value); err == nil {
			t.Errorf("ValidateArtifactURL(%q) accepted untrusted URL", value)
		}
	}
}
