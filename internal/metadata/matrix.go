package metadata

import (
	"encoding/json"
	"fmt"
	"os"
)

// MatrixRow represents a single compatibility matrix entry.
type MatrixRow struct {
	MCVersion    string `json:"mcVersion"`
	JavaMajor    int    `json:"javaMajor"`
	Vanilla      bool   `json:"vanilla"`
	Fabric       bool   `json:"fabric"`
	Quilt        bool   `json:"quilt"`
	PassEvidence string `json:"passEvidence"`
}

// CompatibilityMatrix is a collection of matrix rows.
type CompatibilityMatrix struct {
	Rows []MatrixRow `json:"rows"`
}

// LoadMatrix loads a compatibility matrix from a JSON file.
func LoadMatrix(path string) (*CompatibilityMatrix, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read matrix: %w", err)
	}
	var matrix CompatibilityMatrix
	if err := json.Unmarshal(data, &matrix); err != nil {
		return nil, fmt.Errorf("unmarshal matrix: %w", err)
	}
	return &matrix, nil
}

// IsVersionSupported checks if a version exists in the matrix.
func IsVersionSupported(matrix *CompatibilityMatrix, version string) bool {
	if matrix == nil {
		return false
	}
	for _, row := range matrix.Rows {
		if row.MCVersion == version {
			return true
		}
	}
	return false
}

// ValidateMatrix checks a matrix for consistency.
func ValidateMatrix(matrix *CompatibilityMatrix) error {
	if matrix == nil {
		return fmt.Errorf("matrix is nil")
	}
	seen := make(map[string]bool, len(matrix.Rows))
	for _, row := range matrix.Rows {
		if row.MCVersion == "" {
			return fmt.Errorf("matrix row has no mcVersion")
		}
		if seen[row.MCVersion] {
			return fmt.Errorf("duplicate version: %s", row.MCVersion)
		}
		seen[row.MCVersion] = true
		if row.JavaMajor == 0 {
			return fmt.Errorf("version %s has no javaMajor", row.MCVersion)
		}
		if row.PassEvidence == "" {
			return fmt.Errorf("version %s has no passEvidence", row.MCVersion)
		}
	}
	return nil
}
