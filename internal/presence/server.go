package presence

import (
	"regexp"
	"strings"
)

// ServerEvent is the outcome of parsing one game log line for POST-005
// best-effort server presence.
type ServerEvent int

const (
	// ServerNone means the line carried nothing usable.
	ServerNone ServerEvent = iota
	// ServerConnected means a host (and possibly a port) was found.
	ServerConnected
	// ServerCleared means the player left the server.
	ServerCleared
)

// ServerAddress is a sanitised server host and optional port. An empty
// Host never reaches a payload.
type ServerAddress struct {
	Host string
	Port int
}

// MaxServerAddressLen rejects an unreasonably long host, so a crafted or
// corrupted log line cannot flood Discord's 128-byte state field.
const MaxServerAddressLen = 64

// serverPatterns is the bounded set of connection lines recognised in
// latest.log. Lines outside this set are ignored rather than guessed, which
// is what POST-005 asks for. Host and port are captured separately so the
// caller can decide whether the port is published.
//
// The pattern deliberately rejects IPv6 literals: a bare "::1" cannot be
// split from a ":port" suffix unambiguously, so those lines are omitted
// instead of guessed.
var serverPatterns = []*regexp.Regexp{
	// "Connecting to play.example.com, 25565"  (legacy comma form)
	// "Connecting to play.example.com:25565"   (host:port form)
	// "Connecting to play.example.com"          (no port)
	regexp.MustCompile(`^Connecting to ([^,\s:]+)(?:[:,]\s*(\d{1,5}))?$`),
}

// clearPatterns mark the player leaving a server. They are best-effort:
// when none match, the last server stays published until the game exits.
var clearPatterns = []*regexp.Regexp{
	regexp.MustCompile(`^Disconnecting\b`),
	regexp.MustCompile(`^Disconnected\b`),
}

// ParseServerLine extracts a server address from one game log line. Both
// stdout and latest.log lines are accepted; the optional Log4j2 prefix
// ("[12:34:56] [Render thread/INFO]: ") is stripped first.
func ParseServerLine(line string) (ServerEvent, ServerAddress) {
	message := logLineMessage(line)

	for _, pattern := range clearPatterns {
		if pattern.MatchString(message) {
			return ServerCleared, ServerAddress{}
		}
	}

	for _, pattern := range serverPatterns {
		match := pattern.FindStringSubmatch(message)
		if match == nil {
			continue
		}
		host := sanitizeHost(match[1])
		if host == "" {
			return ServerNone, ServerAddress{}
		}
		address := ServerAddress{Host: host}
		if len(match) > 2 && match[2] != "" {
			// An impossible port means the line is not trustworthy, so
			// the whole address is dropped rather than published host-only.
			port, ok := sanitizePort(match[2])
			if !ok {
				return ServerNone, ServerAddress{}
			}
			address.Port = port
		}
		return ServerConnected, address
	}
	return ServerNone, ServerAddress{}
}

// logLineMessage strips the optional Log4j2 prefix that latest.log writes.
// The prefix only counts when the line actually starts with "[", so plain
// console lines are left untouched.
func logLineMessage(line string) string {
	line = strings.TrimSpace(line)
	if line == "" || line[0] != '[' {
		return line
	}
	if i := strings.Index(line, "]: "); i >= 0 && i < 40 {
		return strings.TrimSpace(line[i+3:])
	}
	return line
}

// sanitizeHost accepts only characters a host may contain. Anything else
// makes the line unrecognised rather than silently rewritten into a
// different host, and an over-long host is dropped instead of truncated
// into a misleading prefix.
func sanitizeHost(raw string) string {
	if raw == "" || len(raw) > MaxServerAddressLen {
		return ""
	}
	for _, r := range raw {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9',
			r == '.', r == '-', r == '_':
		default:
			return ""
		}
	}
	host := strings.Trim(raw, ".-")
	if host == "" {
		return ""
	}
	return host
}

// sanitizePort accepts only a valid TCP port. Out-of-range values are
// dropped rather than published.
func sanitizePort(raw string) (int, bool) {
	port := 0
	for _, r := range raw {
		if r < '0' || r > '9' {
			return 0, false
		}
		port = port*10 + int(r-'0')
		if port > 65535 {
			return 0, false
		}
	}
	if port < 1 {
		return 0, false
	}
	return port, true
}
