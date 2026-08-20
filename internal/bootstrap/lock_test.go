package bootstrap_test

import (
	"testing"

	"plumelauncher/internal/bootstrap"
)

func TestDataRootLockIsExclusiveAndReleasable(t *testing.T) {
	root := t.TempDir()
	first, err := bootstrap.AcquireDataRootLock(root)
	if err != nil {
		t.Fatalf("first lock: %v", err)
	}
	defer first.Release()

	if _, err := bootstrap.AcquireDataRootLock(root); err == nil {
		t.Fatal("second process acquired the lock")
	}
	if err := first.Release(); err != nil {
		t.Fatalf("release: %v", err)
	}
	second, err := bootstrap.AcquireDataRootLock(root)
	if err != nil {
		t.Fatalf("reacquire: %v", err)
	}
	if err := second.Release(); err != nil {
		t.Fatalf("second release: %v", err)
	}
}
