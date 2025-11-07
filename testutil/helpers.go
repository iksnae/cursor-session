package testutil

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// LoadFixture loads a test fixture file
func LoadFixture(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", path))
	if err != nil {
		t.Fatalf("Failed to load fixture %s: %v", path, err)
	}
	return data
}

// WriteFixture writes a test fixture file
func WriteFixture(t *testing.T, path string, data []byte) {
	t.Helper()
	fullPath := filepath.Join("testdata", path)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		t.Fatalf("Failed to create fixture directory: %v", err)
	}
	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		t.Fatalf("Failed to write fixture %s: %v", path, err)
	}
}

// CreateTempDir creates a temporary directory for testing
func CreateTempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "cursor-session-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	return dir
}

// JSONMarshal marshals a value to JSON for testing
func JSONMarshal(t *testing.T, v interface{}) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}
	return data
}

// JSONUnmarshal unmarshals JSON for testing
func JSONUnmarshal(t *testing.T, data []byte, v interface{}) {
	t.Helper()
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}
}
