package auth

import (
	"crypto/md5"
	"fmt"
	"strings"
)

// OfflineUUID generates a deterministic UUID for an offline player.
// Uses the Minecraft algorithm: MD5("OfflinePlayer:<username>").
func OfflineUUID(username string) string {
	hash := md5.Sum([]byte("OfflinePlayer:" + username))
	hash[6] = (hash[6] & 0x0f) | 0x30 // Version 3
	hash[8] = (hash[8] & 0x3f) | 0x80 // RFC 4122 variant

	return fmt.Sprintf("%02x%02x%02x%02x-%02x%02x-%02x%02x-%02x%02x-%02x%02x%02x%02x%02x%02x",
		hash[0], hash[1], hash[2], hash[3],
		hash[4], hash[5],
		hash[6], hash[7],
		hash[8], hash[9],
		hash[10], hash[11], hash[12], hash[13], hash[14], hash[15])
}

// OfflineDisplayName returns a display name for offline accounts.
func OfflineDisplayName(username string) string {
	return username
}

// NormalizeUsername strips spaces and control characters from a username.
func NormalizeUsername(username string) string {
	var b strings.Builder
	for _, r := range username {
		if r >= 32 { // skip control characters
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}
