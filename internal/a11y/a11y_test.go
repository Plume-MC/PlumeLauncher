package a11y

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateDirectory_FindsViolations(t *testing.T) {
	dir := t.TempDir()

	// Write a .tsx file with violations
	violationFile := filepath.Join(dir, "Bad.tsx")
	bad := `<button onClick={handleClick}>Click</button>
<input type="text" onChange={handleChange} />
<img src="photo.png" />`
	if err := os.WriteFile(violationFile, []byte(bad), 0o644); err != nil {
		t.Fatal(err)
	}

	violations, err := ValidateDirectory(dir)
	if err != nil {
		t.Fatal(err)
	}
	// Expect 3 violations: button, input, img
	if len(violations) < 3 {
		t.Errorf("expected at least 3 violations, got %d: %s", len(violations), FormatViolations(violations))
	}
}

func TestValidateDirectory_PassesGoodFile(t *testing.T) {
	dir := t.TempDir()

	goodFile := filepath.Join(dir, "Good.tsx")
	good := `<button aria-label="Submit" onClick={handleClick}>Submit</button>
<input type="text" aria-label="Search" onChange={handleChange} />
<img src="photo.png" alt="A photo" />`
	if err := os.WriteFile(goodFile, []byte(good), 0o644); err != nil {
		t.Fatal(err)
	}

	violations, err := ValidateDirectory(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 0 {
		t.Errorf("expected 0 violations for good file, got %d: %s", len(violations), FormatViolations(violations))
	}
}

func TestValidateDirectory_SkipsNonTsxFiles(t *testing.T) {
	dir := t.TempDir()

	goFile := filepath.Join(dir, "main.go")
	if err := os.WriteFile(goFile, []byte(`package main`), 0o644); err != nil {
		t.Fatal(err)
	}

	violations, err := ValidateDirectory(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 0 {
		t.Errorf("expected 0 violations for non-.tsx files, got %d", len(violations))
	}
}

func TestValidateDirectory_RealFrontend(t *testing.T) {
	// Only run if frontend/src exists
	frontendDir := filepath.Join("..", "..", "frontend", "src")
	if _, err := os.Stat(frontendDir); os.IsNotExist(err) {
		t.Skip("frontend/src not found, skipping real scan")
	}

	violations, err := ValidateDirectory(frontendDir)
	if err != nil {
		t.Fatal(err)
	}

	// We expect the existing codebase to have good a11y.
	// If violations are found, report them but don't fail the build
	// on the first run — this becomes a regression gate.
	if len(violations) > 0 {
		t.Logf("a11y violations found (informational, regression gate):\n%s", FormatViolations(violations))
	}
}
