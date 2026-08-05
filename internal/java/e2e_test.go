package java_test

import (
	"testing"

	"plumelauncher/internal/java"
)

func TestScanJavaInstallations(t *testing.T) {
	installs, err := java.ScanJavaInstallations()
	if err != nil {
		t.Fatalf("ScanJavaInstallations: %v", err)
	}

	t.Logf("Found %d Java installations:", len(installs))
	for _, inst := range installs {
		t.Logf("  Java %d (%s) at %s", inst.Major, inst.Version, inst.Path)
	}

	// Should find at least one Java on the dev machine
	if len(installs) == 0 {
		t.Log("No Java installations found — this is OK if Java is not installed")
	}
}

func TestCheckJavaCurrent(t *testing.T) {
	installs, err := java.ScanJavaInstallations()
	if err != nil {
		t.Fatalf("ScanJavaInstallations: %v", err)
	}
	if len(installs) == 0 {
		t.Skip("No Java installations found")
	}

	// Check the first found Java
	info, err := java.CheckJava(installs[0].Path)
	if err != nil {
		t.Fatalf("CheckJava: %v", err)
	}
	if info == nil {
		t.Fatal("CheckJava returned nil")
	}
	if info.Major < 8 {
		t.Errorf("Major = %d, expected >= 8", info.Major)
	}
}

func TestSelectJavaFromScan(t *testing.T) {
	installs, err := java.ScanJavaInstallations()
	if err != nil {
		t.Fatalf("ScanJavaInstallations: %v", err)
	}
	if len(installs) == 0 {
		t.Skip("No Java installations found")
	}

	// Try to select Java for MC 1.18.2 (needs Java 17)
	result, err := java.SelectJava(installs, "1.18.2")
	if err != nil {
		t.Logf("SelectJava for 1.18.2: %v (expected if no Java 17)", err)
	} else {
		t.Logf("Selected Java %d for MC 1.18.2: %s", result.Major, result.Path)
	}
}

func TestValidateJavaPath(t *testing.T) {
	installs, err := java.ScanJavaInstallations()
	if err != nil {
		t.Fatalf("ScanJavaInstallations: %v", err)
	}
	if len(installs) == 0 {
		t.Skip("No Java installations found")
	}

	// Validate with correct major
	info, err := java.ValidateJavaPath(installs[0].Path, installs[0].Major)
	if err != nil {
		t.Fatalf("ValidateJavaPath: %v", err)
	}
	if info.Major != installs[0].Major {
		t.Errorf("Major = %d, want %d", info.Major, installs[0].Major)
	}
}

func TestValidateJavaPathWrongMajor(t *testing.T) {
	installs, err := java.ScanJavaInstallations()
	if err != nil {
		t.Fatalf("ScanJavaInstallations: %v", err)
	}
	if len(installs) == 0 {
		t.Skip("No Java installations found")
	}

	// Validate with wrong major (999)
	_, err = java.ValidateJavaPath(installs[0].Path, 999)
	if err == nil {
		t.Fatal("expected error for wrong major")
	}
}

func TestValidateJavaPathNotFound(t *testing.T) {
	_, err := java.ValidateJavaPath("/nonexistent/java", 17)
	if err == nil {
		t.Fatal("expected error for nonexistent path")
	}
}
