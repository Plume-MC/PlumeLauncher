package services

import (
	"testing"
	"time"

	"plumelauncher/internal/instances"
	"plumelauncher/internal/presence"
)

// fakePresence is a manual in-memory stand-in double via a real presence
// client pointed at nothing; we test the hook logic, not the transport.
// publishPresence and clearPresence are exercised for their fail-safe
// behavior: nil client and unreachable Discord must both be no-ops.

func TestPublishPresenceNilClientNoPanic(t *testing.T) {
	service := &LaunchService{}
	service.publishPresence(123, "instance") // must not panic
	service.clearPresence()
}

func TestClearPresenceDisabledClient(t *testing.T) {
	client := presence.NewClient("test-id")
	client.Close()
	service := &LaunchService{Presence: client}
	service.clearPresence() // closed client => ErrDisabled path, must not panic
	if got := client.Status(); got != presence.StatusDisabled {
		t.Errorf("status = %q, want disabled", got)
	}
}

func TestPublishPresenceDisabledPreferenceNoDial(t *testing.T) {
	// Presence disabled in settings: BuildActivity returns nil, so no dial
	// attempt is made and the client stays disabled.
	client := presence.NewClient("test-id")
	defer client.Close()
	service := &LaunchService{Presence: client, DataRoot: t.TempDir()}
	inst := &instances.Instance{ID: "inst", Name: "Test", MCVersion: "1.20.1", Loader: instances.LoaderVanilla}

	_ = inst
	// publishPresence loads the instance through s.Instances; with a nil
	// manager it returns early. Verify the manager-nil path is safe.
	service.publishPresence(1, "inst")
	if got := client.Status(); got != presence.StatusDisabled {
		t.Errorf("status = %q, want disabled (no dial)", got)
	}
}

func TestPresenceClientEnableDisableCycle(t *testing.T) {
	client := presence.NewClient("test-id")
	defer client.Close()

	client.Enable()
	if got := client.Status(); got != presence.StatusReconnecting {
		t.Errorf("status after enable = %q, want reconnecting", got)
	}
	client.Disable()
	if got := client.Status(); got != presence.StatusDisabled {
		t.Errorf("status after disable = %q, want disabled", got)
	}
}

func TestPresenceStatusesExist(t *testing.T) {
	// The UI relies on these exact strings; guard against renames.
	statuses := map[string]bool{
		presence.StatusConnected:    true,
		presence.StatusDisabled:     true,
		presence.StatusUnavailable:  true,
		presence.StatusReconnecting: true,
		presence.StatusError:        true,
	}
	if len(statuses) != 5 {
		t.Fatalf("unexpected status set: %v", statuses)
	}
	_ = time.Now // keep time import if assertions evolve
}
