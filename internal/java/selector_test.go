package java_test

import (
	"testing"

	"plumelauncher/internal/java"
)

func TestRequiredJavaMajor(t *testing.T) {
	tests := []struct {
		mcVersion string
		expected  int
	}{
		{"1.0", 8},
		{"1.5.2", 8},
		{"1.7.10", 8},
		{"1.12.2", 8},
		{"1.16.5", 8},
		{"1.17", 17},
		{"1.17.1", 17},
		{"1.18.2", 17},
		{"1.20.4", 17},
		{"1.20.5", 21},
		{"1.20.6", 21},
		{"1.21", 21},
		{"1.21.4", 21},
		{"26.2", 25},
		{"26.3-snapshot-5", 25},
		{"1.20.1-fabric-0.15.0", 17},
		{"1.18.2-quilt", 17},
	}

	for _, tt := range tests {
		t.Run(tt.mcVersion, func(t *testing.T) {
			got := java.RequiredJavaMajor(tt.mcVersion)
			if got != tt.expected {
				t.Errorf("RequiredJavaMajor(%q) = %d, want %d", tt.mcVersion, got, tt.expected)
			}
		})
	}
}

func TestSelectJavaExactMatch(t *testing.T) {
	installs := []java.JavaInfo{
		{Path: "/path/to/java8", Version: "1.8.0_382", Major: 8},
		{Path: "/path/to/java17", Version: "17.0.8", Major: 17},
		{Path: "/path/to/java21", Version: "21.0.3", Major: 21},
	}

	// MC 1.16.5 needs Java 8, exact match preferred
	result, err := java.SelectJava(installs, "1.16.5")
	if err != nil {
		t.Fatalf("SelectJava: %v", err)
	}
	if result.Major != 8 {
		t.Errorf("Major = %d, want 8", result.Major)
	}

	// MC 1.20.5 needs Java 21, exact match preferred
	result, err = java.SelectJava(installs, "1.20.5")
	if err != nil {
		t.Fatalf("SelectJava: %v", err)
	}
	if result.Major != 21 {
		t.Errorf("Major = %d, want 21", result.Major)
	}
}

func TestSelectJavaCompatibleHigher(t *testing.T) {
	installs := []java.JavaInfo{
		{Path: "/path/to/java25", Version: "25.0.3", Major: 25},
	}

	// MC 1.18.2 needs Java 17 minimum, Java 25 is compatible (25 >= 17)
	result, err := java.SelectJava(installs, "1.18.2")
	if err != nil {
		t.Fatalf("SelectJava: %v", err)
	}
	if result.Major != 25 {
		t.Errorf("Major = %d, want 25", result.Major)
	}
}

func TestSelectJavaCompatibleOlderMC(t *testing.T) {
	installs := []java.JavaInfo{
		{Path: "/path/to/java21", Version: "21.0.3", Major: 21},
	}

	// MC 1.7.10 needs Java 8 minimum, Java 21 is compatible (21 >= 8)
	result, err := java.SelectJava(installs, "1.7.10")
	if err != nil {
		t.Fatalf("SelectJava: %v", err)
	}
	if result.Major != 21 {
		t.Errorf("Major = %d, want 21", result.Major)
	}
}

func TestSelectJavaPrefersExactOverHigher(t *testing.T) {
	installs := []java.JavaInfo{
		{Path: "/path/to/java17", Version: "17.0.8", Major: 17},
		{Path: "/path/to/java21", Version: "21.0.3", Major: 21},
	}

	// MC 1.18.2 needs Java 17 minimum, exact match preferred over higher
	result, err := java.SelectJava(installs, "1.18.2")
	if err != nil {
		t.Fatalf("SelectJava: %v", err)
	}
	if result.Major != 17 {
		t.Errorf("Major = %d, want 17 (exact preferred over 21)", result.Major)
	}
}

func TestSelectJavaClosestHigher(t *testing.T) {
	installs := []java.JavaInfo{
		{Path: "/path/to/java21", Version: "21.0.3", Major: 21},
		{Path: "/path/to/java25", Version: "25.0.3", Major: 25},
	}

	// MC 1.18.2 needs Java 17 minimum, closest above preferred (21 over 25)
	result, err := java.SelectJava(installs, "1.18.2")
	if err != nil {
		t.Fatalf("SelectJava: %v", err)
	}
	if result.Major != 21 {
		t.Errorf("Major = %d, want 21 (closest above preferred)", result.Major)
	}
}

func TestSelectJavaNoMatch(t *testing.T) {
	installs := []java.JavaInfo{
		{Path: "/path/to/java8", Version: "1.8.0_382", Major: 8},
	}

	// MC 1.20.5 needs Java 21, only Java 8 available
	_, err := java.SelectJava(installs, "1.20.5")
	if err == nil {
		t.Fatal("expected error for no match")
	}
}

func TestSelectJavaNoInstalls(t *testing.T) {
	_, err := java.SelectJava(nil, "1.18.2")
	if err == nil {
		t.Fatal("expected error for no installs")
	}
}

func TestSelectJavaHighestVersionWins(t *testing.T) {
	installs := []java.JavaInfo{
		{Path: "/path/to/java17a", Version: "17.0.6", Major: 17},
		{Path: "/path/to/java17b", Version: "17.0.8", Major: 17},
	}

	result, err := java.SelectJava(installs, "1.18.2")
	if err != nil {
		t.Fatalf("SelectJava: %v", err)
	}
	if result.Version != "17.0.8" {
		t.Errorf("Version = %q, want %q", result.Version, "17.0.8")
	}
}
