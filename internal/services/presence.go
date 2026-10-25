package services

import "plumelauncher/internal/presence"

// PresenceService exposes Discord Rich Presence status. The enabled flag is
// owned by launcher settings (SystemService.UpdateSettings), so this service
// only reports state.
type PresenceService struct {
	Presence *presence.Client
}

// PresenceStatus reports the current IPC connection state.
func (s *PresenceService) GetPresenceStatus() string {
	if s.Presence == nil {
		return presence.StatusDisabled
	}
	return s.Presence.Status()
}
