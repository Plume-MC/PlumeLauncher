package presence

import "testing"

func TestParseServerLineConnectPatterns(t *testing.T) {
	cases := []struct {
		name string
		line string
		host string
		port int
	}{
		{
			name: "legacy comma form",
			line: "Connecting to play.example.com, 25565",
			host: "play.example.com", port: 25565,
		},
		{
			name: "host port form",
			line: "Connecting to play.example.com:25565",
			host: "play.example.com", port: 25565,
		},
		{
			name: "no port",
			line: "Connecting to play.example.com",
			host: "play.example.com", port: 0,
		},
		{
			name: "log4j prefix",
			line: "[12:34:56] [Render thread/INFO]: Connecting to play.example.com, 25565",
			host: "play.example.com", port: 25565,
		},
		{
			name: "ipv4 address",
			line: "Connecting to 192.168.1.5, 25565",
			host: "192.168.1.5", port: 25565,
		},
		{
			name: "underscore host",
			line: "Connecting to my_server.local:25565",
			host: "my_server.local", port: 25565,
		},
		{
			name: "non standard port",
			line: "Connecting to play.example.com:25566",
			host: "play.example.com", port: 25566,
		},
		{
			name: "trailing spaces",
			line: "  Connecting to play.example.com, 25565  ",
			host: "play.example.com", port: 25565,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			event, address := ParseServerLine(tc.line)
			if event != ServerConnected {
				t.Fatalf("event = %v, want ServerConnected", event)
			}
			if address.Host != tc.host {
				t.Errorf("host = %q, want %q", address.Host, tc.host)
			}
			if address.Port != tc.port {
				t.Errorf("port = %d, want %d", address.Port, tc.port)
			}
		})
	}
}

func TestParseServerLineClearPatterns(t *testing.T) {
	cases := []string{
		"Disconnecting",
		"Disconnected",
		"[12:34:56] [Render thread/INFO]: Disconnecting",
	}
	for _, line := range cases {
		t.Run(line, func(t *testing.T) {
			event, address := ParseServerLine(line)
			if event != ServerCleared {
				t.Fatalf("event = %v, want ServerCleared", event)
			}
			if address.Host != "" {
				t.Errorf("host = %q, want empty", address.Host)
			}
		})
	}
}

func TestParseServerLineUnknownIsOmitted(t *testing.T) {
	// POST-005: unknown values are omitted instead of guessed.
	cases := []string{
		"",
		"Starting integrated minecraft server version 1.21.1",
		"Connecting to unknown server",
		"Connecting to <script>alert(1)</script>, 25565",
		"Connecting to .., 25565",
		"Connecting to play.example.com, 99999", // port out of range
		"Connecting to play.example.com:0",
		"Connecting to " + string(make([]byte, MaxServerAddressLen+1)) + ", 25565",
		"Setting user: Steve",
		"[12:34:56] [Render thread/INFO]: Stopping integrated minecraft server",
	}
	for _, line := range cases {
		t.Run(line, func(t *testing.T) {
			event, address := ParseServerLine(line)
			if event != ServerNone {
				t.Fatalf("event = %v, want ServerNone", event)
			}
			if address.Host != "" {
				t.Errorf("host = %q, want empty", address.Host)
			}
			if address.Port != 0 {
				t.Errorf("port = %d, want 0", address.Port)
			}
		})
	}
}

func TestParseServerLineControlCharactersRejected(t *testing.T) {
	// A control character must never reach a payload, not even one that
	// would otherwise look like a valid host.
	line := "Connecting to play\x00.example.com, 25565"
	event, address := ParseServerLine(line)
	if event != ServerNone {
		t.Fatalf("event = %v, want ServerNone", event)
	}
	if address.Host != "" {
		t.Errorf("host = %q, want empty", address.Host)
	}
}

func TestParseServerLineIsBounded(t *testing.T) {
	// A very long but valid-looking host is dropped, not truncated into a
	// prefix that would publish the wrong address.
	host := make([]byte, MaxServerAddressLen+1)
	for i := range host {
		host[i] = 'a'
	}
	line := "Connecting to " + string(host) + ", 25565"
	event, address := ParseServerLine(line)
	if event != ServerNone {
		t.Fatalf("event = %v, want ServerNone for over-long host", event)
	}
	if address.Host != "" {
		t.Errorf("host = %q, want empty", address.Host)
	}
}
