package instances_test

import (
	"testing"

	"plumelauncher/internal/instances"
)

func TestValidTransitions(t *testing.T) {
	tests := []struct {
		from  instances.InstanceState
		to    instances.InstanceState
		valid bool
	}{
		{instances.StateNotInstalled, instances.StatePlanning, true},
		{instances.StateNotInstalled, instances.StateFailed, true},
		{instances.StateNotInstalled, instances.StateReady, false},
		{instances.StatePlanning, instances.StateDownloading, true},
		{instances.StatePlanning, instances.StateFailed, true},
		{instances.StatePlanning, instances.StateReady, false},
		{instances.StateDownloading, instances.StateVerifying, true},
		{instances.StateDownloading, instances.StateFailed, true},
		{instances.StateVerifying, instances.StateReady, true},
		{instances.StateVerifying, instances.StateFailed, true},
		{instances.StateReady, instances.StateRunning, true},
		{instances.StateReady, instances.StateFailed, false},
		{instances.StateRunning, instances.StateStopped, true},
		{instances.StateRunning, instances.StateCrashed, true},
		{instances.StateRunning, instances.StateFailed, true},
		{instances.StateStopped, instances.StatePlanning, true},
		{instances.StateStopped, instances.StateReady, false},
		{instances.StateCrashed, instances.StatePlanning, true},
		{instances.StateFailed, instances.StatePlanning, true},
		{instances.StateFailed, instances.StateReady, false},
	}

	for _, tt := range tests {
		name := string(tt.from) + "→" + string(tt.to)
		t.Run(name, func(t *testing.T) {
			got := instances.CanTransition(tt.from, tt.to)
			if got != tt.valid {
				t.Errorf("CanTransition(%s, %s) = %v, want %v", tt.from, tt.to, got, tt.valid)
			}
		})
	}
}

func TestTransitionSuccess(t *testing.T) {
	state, err := instances.Transition(instances.StateNotInstalled, instances.StatePlanning)
	if err != nil {
		t.Fatalf("Transition: %v", err)
	}
	if state != instances.StatePlanning {
		t.Errorf("state = %s, want %s", state, instances.StatePlanning)
	}
}

func TestTransitionInvalid(t *testing.T) {
	_, err := instances.Transition(instances.StateNotInstalled, instances.StateReady)
	if err == nil {
		t.Fatal("expected error for invalid transition")
	}
}

func TestInstanceModel(t *testing.T) {
	inst := instances.Instance{
		ID:        "test-123",
		Name:      "Test Instance",
		MCVersion: "1.21.4",
		Loader:    instances.LoaderVanilla,
		State:     instances.StateNotInstalled,
	}

	if inst.ID != "test-123" {
		t.Errorf("ID = %q", inst.ID)
	}
	if inst.State != instances.StateNotInstalled {
		t.Errorf("State = %s", inst.State)
	}
}
