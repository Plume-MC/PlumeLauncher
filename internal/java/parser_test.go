package java_test

import (
	"testing"

	"plumelauncher/internal/java"
)

func TestParseJavaVersion(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{"openjdk17", `openjdk version "17.0.8" 2023-07-18`, "17.0.8"},
		{"openjdk21", `openjdk version "21.0.3" 2024-01-16`, "21.0.3"},
		{"oracle8", `java version "1.8.0_382"`, "1.8.0_382"},
		{"corretto17", `openjdk version "17.0.8.7-Amazon-corretto"`, "17.0.8.7-Amazon-corretto"},
		{"empty", "", ""},
		{"noVersion", `some random output`, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := java.ParseJavaVersion(tt.input)
			if got != tt.expect {
				t.Errorf("ParseJavaVersion(%q) = %q, want %q", tt.input, got, tt.expect)
			}
		})
	}
}

func TestParseMajorVersion(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect int
	}{
		{"legacy8", "1.8.0_382", 8},
		{"modern17", "17.0.8", 17},
		{"modern21", "21.0.3", 21},
		{"modern25", "25.0.0", 25},
		{"withEA", "17.0.8-ea", 17},
		{"withPlus", "17.0.8+1", 17},
		{"justNumber", "21", 21},
		{"empty", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := java.ParseMajorVersion(tt.input)
			if got != tt.expect {
				t.Errorf("ParseMajorVersion(%q) = %d, want %d", tt.input, got, tt.expect)
			}
		})
	}
}

func TestParseMajorVersionRoundtrip(t *testing.T) {
	output := `openjdk version "17.0.8" 2023-07-18
OpenJDK Runtime Environment (build 17.0.8+7-post-Ubuntu-0ubuntu122.04.1)
OpenJDK 64-Bit Server VM (build 17.0.8+7-post-Ubuntu-0ubuntu122.04.1, mixed mode, sharing)`

	version := java.ParseJavaVersion(output)
	major := java.ParseMajorVersion(version)

	if version != "17.0.8" {
		t.Errorf("version = %q, want %q", version, "17.0.8")
	}
	if major != 17 {
		t.Errorf("major = %d, want 17", major)
	}
}
