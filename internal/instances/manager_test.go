package instances_test

import (
	"os"
	"path/filepath"
	"testing"

	"plumelauncher/internal/instances"
)

func TestCreateInstance(t *testing.T) {
	dir := t.TempDir()
	defaults := instances.DefaultLauncherDefaults()
	mgr := instances.NewManager(dir, defaults)

	inst, err := mgr.Create("Test Instance", "1.21.4", instances.LoaderVanilla)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if inst.Name != "Test Instance" {
		t.Errorf("Name = %q", inst.Name)
	}
	if inst.MCVersion != "1.21.4" {
		t.Errorf("MCVersion = %q", inst.MCVersion)
	}
	if inst.State != instances.StateNotInstalled {
		t.Errorf("State = %s", inst.State)
	}
	if inst.ID == "" {
		t.Error("ID is empty")
	}
	if _, err := os.Stat(filepath.Join(dir, "instances", inst.ID, ".minecraft")); err != nil {
		t.Errorf("isolated game directory: %v", err)
	}
}

func TestCreateInstanceEmptyName(t *testing.T) {
	dir := t.TempDir()
	mgr := instances.NewManager(dir, instances.DefaultLauncherDefaults())

	_, err := mgr.Create("", "1.21.4", instances.LoaderVanilla)
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestGetInstance(t *testing.T) {
	dir := t.TempDir()
	mgr := instances.NewManager(dir, instances.DefaultLauncherDefaults())

	created, _ := mgr.Create("Test", "1.21.4", instances.LoaderVanilla)

	got, err := mgr.Get(created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != "Test" {
		t.Errorf("Name = %q, want %q", got.Name, "Test")
	}
}

func TestGetInstanceNotFound(t *testing.T) {
	dir := t.TempDir()
	mgr := instances.NewManager(dir, instances.DefaultLauncherDefaults())

	_, err := mgr.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent instance")
	}
}

func TestListInstances(t *testing.T) {
	dir := t.TempDir()
	mgr := instances.NewManager(dir, instances.DefaultLauncherDefaults())

	mgr.Create("Instance 1", "1.21.4", instances.LoaderVanilla)
	mgr.Create("Instance 2", "1.18.2", instances.LoaderFabric)

	list, err := mgr.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("List len = %d, want 2", len(list))
	}
}

func TestListInstancesEmpty(t *testing.T) {
	dir := t.TempDir()
	mgr := instances.NewManager(dir, instances.DefaultLauncherDefaults())

	list, err := mgr.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("List len = %d, want 0", len(list))
	}
}

func TestDeleteInstance(t *testing.T) {
	dir := t.TempDir()
	mgr := instances.NewManager(dir, instances.DefaultLauncherDefaults())

	created, _ := mgr.Create("Test", "1.21.4", instances.LoaderVanilla)

	err := mgr.Delete(created.ID)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err = mgr.Get(created.ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestDeleteInstanceNotFound(t *testing.T) {
	dir := t.TempDir()
	mgr := instances.NewManager(dir, instances.DefaultLauncherDefaults())

	err := mgr.Delete("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent instance")
	}
}

func TestUpdateState(t *testing.T) {
	dir := t.TempDir()
	mgr := instances.NewManager(dir, instances.DefaultLauncherDefaults())

	created, _ := mgr.Create("Test", "1.21.4", instances.LoaderVanilla)

	err := mgr.UpdateState(created.ID, instances.StatePlanning)
	if err != nil {
		t.Fatalf("UpdateState: %v", err)
	}

	got, _ := mgr.Get(created.ID)
	if got.State != instances.StatePlanning {
		t.Errorf("State = %s, want %s", got.State, instances.StatePlanning)
	}
}

func TestUpdateStateInvalid(t *testing.T) {
	dir := t.TempDir()
	mgr := instances.NewManager(dir, instances.DefaultLauncherDefaults())

	created, _ := mgr.Create("Test", "1.21.4", instances.LoaderVanilla)

	err := mgr.UpdateState(created.ID, instances.StateReady)
	if err == nil {
		t.Fatal("expected error for invalid transition")
	}
}

func TestUpdateSettings(t *testing.T) {
	dir := t.TempDir()
	mgr := instances.NewManager(dir, instances.DefaultLauncherDefaults())

	created, _ := mgr.Create("Test", "1.21.4", instances.LoaderVanilla)

	settings := instances.Settings{
		MinRamMB: instances.IntPtr(2048),
		MaxRamMB: instances.IntPtr(8192),
	}

	err := mgr.UpdateSettings(created.ID, settings)
	if err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}

	got, _ := mgr.Get(created.ID)
	if got.Settings.MinRamMB == nil || *got.Settings.MinRamMB != 2048 {
		t.Error("MinRamMB not updated")
	}
}

func TestInstancePersistence(t *testing.T) {
	dir := t.TempDir()
	defaults := instances.DefaultLauncherDefaults()

	// Create with one manager
	mgr1 := instances.NewManager(dir, defaults)
	created, _ := mgr1.Create("Persistent", "1.21.4", instances.LoaderVanilla)
	mgr1.UpdateState(created.ID, instances.StatePlanning)

	// Load with another manager (simulates restart)
	mgr2 := instances.NewManager(dir, defaults)
	got, err := mgr2.Get(created.ID)
	if err != nil {
		t.Fatalf("Get after restart: %v", err)
	}
	if got.State != instances.StatePlanning {
		t.Errorf("State = %s, want %s", got.State, instances.StatePlanning)
	}
	if got.Name != "Persistent" {
		t.Errorf("Name = %q, want %q", got.Name, "Persistent")
	}
}
