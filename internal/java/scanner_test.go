package java

import "testing"

func TestScanJavaInstallationsCachesSuccessfulScan(t *testing.T) {
	javaScanMu.Lock()
	originalScanner := scanJavaInstallations
	originalCache := cachedJavaInstalls
	originalCached := cachedJavaScan
	cachedJavaInstalls = nil
	cachedJavaScan = false
	calls := 0
	scanJavaInstallations = func() ([]JavaInfo, error) {
		calls++
		return []JavaInfo{{Path: "java", Major: 21}}, nil
	}
	javaScanMu.Unlock()
	t.Cleanup(func() {
		javaScanMu.Lock()
		scanJavaInstallations = originalScanner
		cachedJavaInstalls = originalCache
		cachedJavaScan = originalCached
		javaScanMu.Unlock()
	})

	installs, err := ScanJavaInstallations()
	if err != nil {
		t.Fatalf("ScanJavaInstallations: %v", err)
	}
	installs[0].Path = "changed"

	installs, err = ScanJavaInstallations()
	if err != nil {
		t.Fatalf("ScanJavaInstallations cached: %v", err)
	}
	if calls != 1 || installs[0].Path != "java" {
		t.Fatalf("cached scan = %#v after %d calls, want java after 1 call", installs, calls)
	}

	if _, err := RescanJavaInstallations(); err != nil {
		t.Fatalf("RescanJavaInstallations: %v", err)
	}
	if calls != 2 {
		t.Fatalf("rescan calls = %d, want 2", calls)
	}
}
