package instances_test

import (
	"testing"

	"plumelauncher/internal/instances"
)

func TestRegistryStartOperation(t *testing.T) {
	reg := instances.NewRegistry()

	op, err := reg.Start("inst-1", instances.OpDownload)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if op.Status != instances.OpStatusRunning {
		t.Errorf("Status = %s, want %s", op.Status, instances.OpStatusRunning)
	}
	if !reg.IsActive("inst-1") {
		t.Error("expected active")
	}
}

func TestRegistryDuplicateOperation(t *testing.T) {
	reg := instances.NewRegistry()

	reg.Start("inst-1", instances.OpDownload)
	_, err := reg.Start("inst-1", instances.OpLaunch)
	if err == nil {
		t.Fatal("expected error for duplicate operation")
	}
}

func TestRegistryComplete(t *testing.T) {
	reg := instances.NewRegistry()
	reg.Start("inst-1", instances.OpDownload)

	reg.Complete("inst-1")

	if reg.IsActive("inst-1") {
		t.Error("should not be active after complete")
	}
}

func TestRegistryFail(t *testing.T) {
	reg := instances.NewRegistry()
	reg.Start("inst-1", instances.OpDownload)

	reg.Fail("inst-1")

	if reg.IsActive("inst-1") {
		t.Error("should not be active after fail")
	}
}

func TestRegistryCancel(t *testing.T) {
	reg := instances.NewRegistry()
	op, _ := reg.Start("inst-1", instances.OpDownload)

	reg.Cancel("inst-1")

	if reg.IsActive("inst-1") {
		t.Error("should not be active after cancel")
	}
	// Context should be cancelled
	select {
	case <-op.CancelContext.Done():
		// OK
	default:
		t.Error("context should be cancelled")
	}
}

func TestRegistryGet(t *testing.T) {
	reg := instances.NewRegistry()
	reg.Start("inst-1", instances.OpDownload)

	op := reg.Get("inst-1")
	if op == nil {
		t.Fatal("expected operation")
	}
	if op.InstanceID != "inst-1" {
		t.Errorf("InstanceID = %q", op.InstanceID)
	}
}

func TestRegistryGetNonexistent(t *testing.T) {
	reg := instances.NewRegistry()

	op := reg.Get("nonexistent")
	if op != nil {
		t.Error("expected nil for nonexistent")
	}
}

func TestRegistryDifferentInstances(t *testing.T) {
	reg := instances.NewRegistry()

	reg.Start("inst-1", instances.OpDownload)
	reg.Start("inst-2", instances.OpLaunch)

	if !reg.IsActive("inst-1") {
		t.Error("inst-1 should be active")
	}
	if !reg.IsActive("inst-2") {
		t.Error("inst-2 should be active")
	}

	reg.Complete("inst-1")

	if reg.IsActive("inst-1") {
		t.Error("inst-1 should not be active")
	}
	if !reg.IsActive("inst-2") {
		t.Error("inst-2 should still be active")
	}
}

func TestRegistryRestartAfterComplete(t *testing.T) {
	reg := instances.NewRegistry()

	reg.Start("inst-1", instances.OpDownload)
	reg.Complete("inst-1")

	// Should be able to start a new operation after completion
	_, err := reg.Start("inst-1", instances.OpLaunch)
	if err != nil {
		t.Fatalf("Start after complete: %v", err)
	}
}
