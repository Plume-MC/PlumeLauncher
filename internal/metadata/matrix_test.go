package metadata_test

import (
	"os"
	"path/filepath"
	"testing"

	"plumelauncher/internal/metadata"
)

func TestLoadMatrixValid(t *testing.T) {
	fixture := `{"rows":[
		{"mcVersion":"1.0","javaMajor":8,"vanilla":true,"fabric":false,"quilt":false,"passEvidence":"smoke test"},
		{"mcVersion":"1.18.2","javaMajor":17,"vanilla":true,"fabric":true,"quilt":false,"passEvidence":"smoke test"}
	]}`
	path := filepath.Join(t.TempDir(), "matrix.json")
	os.WriteFile(path, []byte(fixture), 0o644)

	matrix, err := metadata.LoadMatrix(path)
	if err != nil {
		t.Fatalf("LoadMatrix: %v", err)
	}
	if len(matrix.Rows) != 2 {
		t.Errorf("Rows len = %d, want 2", len(matrix.Rows))
	}
}

func TestLoadMatrixMissingField(t *testing.T) {
	fixture := `{"rows":[{"mcVersion":"1.0"}]}`
	path := filepath.Join(t.TempDir(), "matrix.json")
	os.WriteFile(path, []byte(fixture), 0o644)

	_, err := metadata.LoadMatrix(path)
	// Should succeed since Go zero-values are allowed
	if err != nil {
		t.Fatalf("LoadMatrix: %v", err)
	}
}

func TestIsVersionSupported(t *testing.T) {
	matrix := &metadata.CompatibilityMatrix{
		Rows: []metadata.MatrixRow{
			{MCVersion: "1.0", JavaMajor: 8},
			{MCVersion: "1.18.2", JavaMajor: 17},
			{MCVersion: "26.2", JavaMajor: 25},
		},
	}

	if !metadata.IsVersionSupported(matrix, "1.0") {
		t.Error("1.0 should be supported")
	}
	if !metadata.IsVersionSupported(matrix, "26.2") {
		t.Error("26.2 should be supported")
	}
	if metadata.IsVersionSupported(matrix, "99.99") {
		t.Error("99.99 should not be supported")
	}
}

func TestValidateMatrixDuplicate(t *testing.T) {
	matrix := &metadata.CompatibilityMatrix{
		Rows: []metadata.MatrixRow{
			{MCVersion: "1.0", JavaMajor: 8, PassEvidence: "test"},
			{MCVersion: "1.0", JavaMajor: 8, PassEvidence: "test"},
		},
	}
	err := metadata.ValidateMatrix(matrix)
	if err == nil {
		t.Fatal("expected error for duplicate version")
	}
}

func TestValidateMatrixMissingJavaMajor(t *testing.T) {
	matrix := &metadata.CompatibilityMatrix{
		Rows: []metadata.MatrixRow{
			{MCVersion: "1.0", PassEvidence: "test"},
		},
	}
	err := metadata.ValidateMatrix(matrix)
	if err == nil {
		t.Fatal("expected error for missing javaMajor")
	}
}

func TestValidateMatrixMissingEvidence(t *testing.T) {
	matrix := &metadata.CompatibilityMatrix{
		Rows: []metadata.MatrixRow{
			{MCVersion: "1.0", JavaMajor: 8},
		},
	}
	err := metadata.ValidateMatrix(matrix)
	if err == nil {
		t.Fatal("expected error for missing passEvidence")
	}
}

func TestValidateMatrixValid(t *testing.T) {
	matrix := &metadata.CompatibilityMatrix{
		Rows: []metadata.MatrixRow{
			{MCVersion: "1.0", JavaMajor: 8, PassEvidence: "smoke"},
			{MCVersion: "1.18.2", JavaMajor: 17, PassEvidence: "smoke"},
		},
	}
	err := metadata.ValidateMatrix(matrix)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
