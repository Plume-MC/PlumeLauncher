package services

import (
	"plumelauncher/internal/instances"
	"plumelauncher/internal/presence"
)

// PresenceService exposes Discord Rich Presence status and controls.
type PresenceService struct {
	DataRoot string
	Defaults *instances.LauncherDefaults
	Presence *presence.Client
}

// PresenceStatus reports the current IPC connection state.
func (s *PresenceService) GetPresenceStatus() string {
	if s.Presence == nil {
		return presence.StatusDisabled
	}
	return s.Presence.Status()
}

// SetPresenceEnabled toggles presence and persists the preference. The
// enabled state takes effect on the next launch; the connection itself is
// established lazily when a game starts.
func (s *PresenceService) SetPresenceEnabled(enabled bool) error {
	defaults, err := instances.LoadConfig(s.DataRoot)
	if err != nil {
		return NewInternalError("Unable to load launcher settings.")
	}
	defaults.DiscordPresenceEnabled = enabled
	if err := instances.SaveConfig(s.DataRoot, defaults); err != nil {
		return NewInternalError("Unable to save launcher settings.")
	}
	if s.Defaults != nil {
		s.Defaults.DiscordPresenceEnabled = enabled
	}
	if s.Presence != nil {
		if enabled {
			s.Presence.Enable()
		} else {
			s.Presence.Disable()
		}
	}
	return nil
}
