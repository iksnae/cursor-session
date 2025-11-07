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

