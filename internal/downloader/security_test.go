package downloader

import "testing"

func TestIsHexHash(t *testing.T) {
	if !isHexHash("0123456789abcdef0123456789abcdef01234567") {
		t.Fatal("valid hash rejected")
	}
	for _, value := range []string{
		"0123456789abcdef0123456789abcdef0123456/",
		"0123456789abcdef0123456789abcdef0123456.",
		"0123456789abcdef0123456789abcdef0123456g",
	} {
		if isHexHash(value) {
			t.Fatalf("invalid hash accepted: %q", value)
		}
	}
}
