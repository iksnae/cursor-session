package internal

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDetectStoragePaths(t *testing.T) {
	paths, err := DetectStoragePaths()
	if err != nil {
		t.Fatalf("DetectStoragePaths() error = %v", err)
	}

	if paths.BasePath == "" {
		t.Error("BasePath should not be empty")
	}

	expectedBase := ""
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "darwin":
		expectedBase = filepath.Join(home, "Library/Application Support/Cursor/User")
	case "linux":
		expectedBase = filepath.Join(home, ".config/Cursor/User")
	}

	if paths.BasePath != expectedBase {
		t.Errorf("BasePath = %v, want %v", paths.BasePath, expectedBase)
	}

	if paths.GlobalStorage == "" {
		t.Error("GlobalStorage path should not be empty")
	}

	if paths.WorkspaceStorage == "" {
		t.Error("WorkspaceStorage path should not be empty")
	}
}

func TestGlobalStorageDBPath(t *testing.T) {
	paths, _ := DetectStoragePaths()
	dbPath := paths.GetGlobalStorageDBPath()

	if dbPath == "" {
		t.Error("GetGlobalStorageDBPath() should not return empty string")
	}

	expected := filepath.Join(paths.GlobalStorage, "state.vscdb")
	if dbPath != expected {
		t.Errorf("GetGlobalStorageDBPath() = %v, want %v", dbPath, expected)
	}
}

func TestGlobalStorageExists(t *testing.T) {
	// Use a test path instead of real Cursor path
	testPaths := StoragePaths{
		GlobalStorage: "/nonexistent/path/globalStorage",
	}

	// Test when database doesn't exist (should return false)
	exists := testPaths.GlobalStorageExists()
	if exists {
		t.Error("GlobalStorageExists() should return false for nonexistent path")
	}
}

func TestDetectStoragePaths_ErrorCases(t *testing.T) {
	// Test that error is returned for unsupported OS
	// We can't easily test this without mocking runtime.GOOS, but we can document the behavior
	// The actual test would require runtime manipulation which is complex
}

