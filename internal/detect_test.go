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

func TestDetectStoragePaths_AgentStoragePath(t *testing.T) {
	paths, err := DetectStoragePaths()
	if err != nil {
		t.Fatalf("DetectStoragePaths() error = %v", err)
	}

	switch runtime.GOOS {
	case "linux":
		// On Linux, AgentStoragePath should be set
		home, _ := os.UserHomeDir()
		expected := filepath.Join(home, ".cursor/chats")
		if paths.AgentStoragePath != expected {
			t.Errorf("AgentStoragePath = %v, want %v", paths.AgentStoragePath, expected)
		}
	case "darwin":
		// On macOS, AgentStoragePath should be empty
		if paths.AgentStoragePath != "" {
			t.Errorf("AgentStoragePath = %v, want empty string on macOS", paths.AgentStoragePath)
		}
	}
}

func TestHasAgentStorage(t *testing.T) {
	paths, _ := DetectStoragePaths()

	// Test with nonexistent path
	testPaths := StoragePaths{
		AgentStoragePath: "/nonexistent/path/.cursor/chats",
	}
	exists := testPaths.HasAgentStorage()
	if exists {
		t.Error("HasAgentStorage() should return false for nonexistent path")
	}

	// Test with empty path (macOS case)
	testPaths.AgentStoragePath = ""
	exists = testPaths.HasAgentStorage()
	if exists {
		t.Error("HasAgentStorage() should return false for empty path")
	}

	// Test with actual path (if it exists)
	if paths.AgentStoragePath != "" {
		exists = paths.HasAgentStorage()
		// This will be true or false depending on whether the directory exists
		// We just verify it doesn't panic
		_ = exists
	}
}

func TestFindAgentStoreDBs(t *testing.T) {
	paths, _ := DetectStoragePaths()

	// Test with nonexistent path
	testPaths := StoragePaths{
		AgentStoragePath: "/nonexistent/path/.cursor/chats",
	}
	storeDBs, err := testPaths.FindAgentStoreDBs()
	if err != nil {
		t.Errorf("FindAgentStoreDBs() error = %v, want nil", err)
	}
	if len(storeDBs) > 0 {
		t.Error("FindAgentStoreDBs() should return empty slice for nonexistent path")
	}

	// Test with empty path
	testPaths.AgentStoragePath = ""
	storeDBs, err = testPaths.FindAgentStoreDBs()
	if err != nil {
		t.Errorf("FindAgentStoreDBs() error = %v, want nil", err)
	}
	if len(storeDBs) > 0 {
		t.Error("FindAgentStoreDBs() should return nil for empty path")
	}

	// Test with actual path (if it exists)
	if paths.AgentStoragePath != "" && paths.HasAgentStorage() {
		storeDBs, err = paths.FindAgentStoreDBs()
		if err != nil {
			t.Errorf("FindAgentStoreDBs() error = %v", err)
		}
		// Just verify it doesn't panic and returns a slice (may be empty)
		if storeDBs == nil {
			t.Error("FindAgentStoreDBs() should return a slice, not nil")
		}
	}
}
