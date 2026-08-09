package instances_test

import (
	"testing"

	"plumelauncher/internal/instances"
)

func TestOpenInstanceFolderValidation(t *testing.T) {
	dir := t.TempDir()

	// Test path traversal rejection
	err := instances.OpenInstanceFolder(dir, "../../etc")
	if err == nil {
		t.Fatal("expected path traversal error")
	}
}

func TestOpenInstanceFolderNotFound(t *testing.T) {
	dir := t.TempDir()

	err := instances.OpenInstanceFolder(dir, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent instance")
	}
}

func TestE2ECreateListDelete(t *testing.T) {
	dir := t.TempDir()
	defaults := instances.DefaultLauncherDefaults()
	mgr := instances.NewManager(dir, defaults)

	// Create instances
	inst1, err := mgr.Create("Instance 1", "1.21.4", instances.LoaderVanilla)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	inst2, err := mgr.Create("Instance 2", "1.18.2", instances.LoaderFabric)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// List
	list, err := mgr.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("List len = %d, want 2", len(list))
	}

	// Get
	got, err := mgr.Get(inst1.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != "Instance 1" {
		t.Errorf("Name = %q", got.Name)
	}

	// Update state
	err = mgr.UpdateState(inst1.ID, instances.StatePlanning)
	if err != nil {
		t.Fatalf("UpdateState: %v", err)
	}
	got, _ = mgr.Get(inst1.ID)
	if got.State != instances.StatePlanning {
		t.Errorf("State = %s", got.State)
	}

	// Delete
	err = mgr.Delete(inst2.ID)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	list, _ = mgr.List()
	if len(list) != 1 {
		t.Errorf("List len = %d, want 1", len(list))
	}
}

func TestE2EStateTransitions(t *testing.T) {
	dir := t.TempDir()
	mgr := instances.NewManager(dir, instances.DefaultLauncherDefaults())

	inst, _ := mgr.Create("Test", "1.21.4", instances.LoaderVanilla)

	// Full lifecycle
	transitions := []instances.InstanceState{
		instances.StatePlanning,
		instances.StateDownloading,
		instances.StateVerifying,
		instances.StateReady,
		instances.StateRunning,
		instances.StateStopped,
	}

	for _, target := range transitions {
		err := mgr.UpdateState(inst.ID, target)
		if err != nil {
			t.Fatalf("UpdateState to %s: %v", target, err)
		}
	}

	got, _ := mgr.Get(inst.ID)
	if got.State != instances.StateStopped {
		t.Errorf("Final state = %s, want %s", got.State, instances.StateStopped)
	}
}

func TestE2ESettingsInheritance(t *testing.T) {
	dir := t.TempDir()
	defaults := instances.DefaultLauncherDefaults()
	defaults.DefaultMinRamMB = 2048
	defaults.DefaultMaxRamMB = 8192
	mgr := instances.NewManager(dir, defaults)

	// Create instance with no overrides
	inst, _ := mgr.Create("Test", "1.21.4", instances.LoaderVanilla)

	// Check effective settings
	effective := instances.EffectiveSettings(*inst, defaults)
	if effective.MinRamMB != 2048 {
		t.Errorf("MinRamMB = %d, want 2048 (inherit)", effective.MinRamMB)
	}
	if effective.MaxRamMB != 8192 {
		t.Errorf("MaxRamMB = %d, want 8192 (inherit)", effective.MaxRamMB)
	}

	// Override
	mgr.UpdateSettings(inst.ID, instances.Settings{
		MinRamMB: instances.IntPtr(4096),
	})

	got, _ := mgr.Get(inst.ID)
	effective = instances.EffectiveSettings(*got, defaults)
	if effective.MinRamMB != 4096 {
		t.Errorf("MinRamMB = %d, want 4096 (override)", effective.MinRamMB)
	}
	if effective.MaxRamMB != 8192 {
		t.Errorf("MaxRamMB = %d, want 8192 (inherit)", effective.MaxRamMB)
	}
}

func TestE2EPersistenceAcrossManagers(t *testing.T) {
	dir := t.TempDir()
	defaults := instances.DefaultLauncherDefaults()

	// Create with manager 1
	mgr1 := instances.NewManager(dir, defaults)
	inst, _ := mgr1.Create("Persistent", "1.21.4", instances.LoaderVanilla)
	mgr1.UpdateState(inst.ID, instances.StatePlanning)
	mgr1.UpdateSettings(inst.ID, instances.Settings{
		MinRamMB: instances.IntPtr(2048),
	})

	// Load with manager 2 (simulates restart)
	mgr2 := instances.NewManager(dir, defaults)
	got, err := mgr2.Get(inst.ID)
	if err != nil {
		t.Fatalf("Get after restart: %v", err)
	}

	if got.State != instances.StatePlanning {
		t.Errorf("State = %s, want %s", got.State, instances.StatePlanning)
	}
	if got.Settings.MinRamMB == nil || *got.Settings.MinRamMB != 2048 {
		t.Error("Settings not persisted")
	}
}

func TestE2EConcurrentOperations(t *testing.T) {
	dir := t.TempDir()
	mgr := instances.NewManager(dir, instances.DefaultLauncherDefaults())
	reg := instances.NewRegistry()

	inst1, _ := mgr.Create("Instance 1", "1.21.4", instances.LoaderVanilla)
	inst2, _ := mgr.Create("Instance 2", "1.18.2", instances.LoaderFabric)

	// Start operations on different instances
	_, err := reg.Start(inst1.ID, instances.OpDownload)
	if err != nil {
		t.Fatalf("Start op1: %v", err)
	}
	_, err = reg.Start(inst2.ID, instances.OpLaunch)
	if err != nil {
		t.Fatalf("Start op2: %v", err)
	}

	// Both should be active
	if !reg.IsActive(inst1.ID) {
		t.Error("inst1 should be active")
	}
	if !reg.IsActive(inst2.ID) {
		t.Error("inst2 should be active")
	}

	// Complete one
	reg.Complete(inst1.ID)

	if reg.IsActive(inst1.ID) {
		t.Error("inst1 should not be active")
	}
	if !reg.IsActive(inst2.ID) {
		t.Error("inst2 should still be active")
	}
}
