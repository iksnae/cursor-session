package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// StoragePaths holds the detected paths for Cursor storage
type StoragePaths struct {
	WorkspaceStorage string // workspaceStorage directory
	GlobalStorage    string // globalStorage directory (modern format)
	BasePath         string // Base Cursor User directory
}

// DetectStoragePaths detects the Cursor storage paths based on the operating system
func DetectStoragePaths() (StoragePaths, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return StoragePaths{}, fmt.Errorf("failed to get home directory: %w", err)
	}

	var basePath string
	switch runtime.GOOS {
	case "darwin":
		basePath = filepath.Join(home, "Library/Application Support/Cursor/User")
	case "linux":
		basePath = filepath.Join(home, ".config/Cursor/User")
	default:
		return StoragePaths{}, fmt.Errorf("unsupported OS: %s (only macOS and Linux are supported)", runtime.GOOS)
	}

	return StoragePaths{
		WorkspaceStorage: filepath.Join(basePath, "workspaceStorage"),
		GlobalStorage:    filepath.Join(basePath, "globalStorage"),
		BasePath:         basePath,
	}, nil
}

// GetGlobalStorageDBPath returns the path to the globalStorage state.vscdb file
func (sp StoragePaths) GetGlobalStorageDBPath() string {
	return filepath.Join(sp.GlobalStorage, "state.vscdb")
}

// GlobalStorageExists checks if the globalStorage database exists
func (sp StoragePaths) GlobalStorageExists() bool {
	dbPath := sp.GetGlobalStorageDBPath()
	_, err := os.Stat(dbPath)
	return err == nil
}
