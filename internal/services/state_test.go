package services

import (
	"testing"

	"plumelauncher/internal/instances"
)

func TestTransitionInstanceStatePersistsTypedLifecycleTransition(t *testing.T) {
	mgr := instances.NewManager(t.TempDir(), instances.DefaultLauncherDefaults())
	inst, err := mgr.Create("Test", "1.21.4", instances.LoaderVanilla)
	if err != nil {
		t.Fatal(err)
	}

	if err := transitionInstanceState(nil, mgr, inst.ID, instances.StatePlanning, "download-1"); err != nil {
		t.Fatalf("transitionInstanceState: %v", err)
	}
	updated, err := mgr.Get(inst.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.State != instances.StatePlanning {
		t.Fatalf("state = %q, want %q", updated.State, instances.StatePlanning)
	}

	err = transitionInstanceState(nil, mgr, inst.ID, instances.StateReady, "download-1")
	serviceErr, ok := err.(*ServiceError)
	if !ok || serviceErr.Code != ErrCodeConflict {
		t.Fatalf("invalid transition error = %#v, want conflict ServiceError", err)
	}
}
