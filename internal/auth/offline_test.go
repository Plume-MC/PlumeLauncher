package auth_test

import (
	"testing"

	"plumelauncher/internal/auth"
)

func TestOfflineUUID(t *testing.T) {
	tests := []struct {
		username string
		expected string
	}{
		{"Notch", "b50ad385-829d-3141-a216-7e7d7539ba7f"},
		{"jeb_", "a762f560-4fce-3236-812a-b80efff0b62b"},
		{"Dinnerbone", "4d258a81-2358-3084-8166-05b9faccad80"},
		{"Player", "a01e3843-e521-3998-958a-f459800e4d11"},
	}

	for _, tt := range tests {
		t.Run(tt.username, func(t *testing.T) {
			got := auth.OfflineUUID(tt.username)
			if got != tt.expected {
				t.Errorf("OfflineUUID(%q) = %q, want %q", tt.username, got, tt.expected)
			}
		})
	}
}

func TestOfflineUUIDDeterministic(t *testing.T) {
	uuid1 := auth.OfflineUUID("TestUser")
	uuid2 := auth.OfflineUUID("TestUser")
	if uuid1 != uuid2 {
		t.Errorf("UUIDs not deterministic: %q != %q", uuid1, uuid2)
	}
}

func TestOfflineUUIDFormat(t *testing.T) {
	uuid := auth.OfflineUUID("test")
	// Should be 36 chars: 8-4-4-4-12
	if len(uuid) != 36 {
		t.Errorf("UUID length = %d, want 36", len(uuid))
	}
	if uuid[8] != '-' || uuid[13] != '-' || uuid[18] != '-' || uuid[23] != '-' {
		t.Errorf("UUID format wrong: %q", uuid)
	}
	// Version should be 3
	if uuid[14] != '3' {
		t.Errorf("UUID version = %c, want 3", uuid[14])
	}
}

func TestNormalizeUsername(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello", "Hello"},
		{"  Hello  ", "Hello"},
		{"Hello\x00World", "HelloWorld"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := auth.NormalizeUsername(tt.input)
			if got != tt.expected {
				t.Errorf("NormalizeUsername(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
