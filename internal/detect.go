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
	AgentStoragePath string // cursor-agent CLI storage directory (~/.cursor/chats/)
}

// DetectStoragePaths detects the Cursor storage paths based on the operating system
func DetectStoragePaths() (StoragePaths, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return StoragePaths{}, fmt.Errorf("failed to get home directory: %w", err)
	}

	var basePath string
	var agentStoragePath string
	switch runtime.GOOS {
	case "darwin":
		basePath = filepath.Join(home, "Library/Application Support/Cursor/User")
		// Agent storage is Linux-only
		agentStoragePath = ""
	case "linux":
		basePath = filepath.Join(home, ".config/Cursor/User")
		// Check both possible locations for cursor-agent storage
		// Priority: .config/cursor/chats (newer location) then .cursor/chats (older location)
		configCursorChats := filepath.Join(home, ".config/cursor/chats")
		dotCursorChats := filepath.Join(home, ".cursor/chats")
		
		if info, err := os.Stat(configCursorChats); err == nil && info.IsDir() {
			agentStoragePath = configCursorChats
		} else if info, err := os.Stat(dotCursorChats); err == nil && info.IsDir() {
			agentStoragePath = dotCursorChats
		} else {
			// Default to .cursor/chats if neither exists (for backward compatibility)
			agentStoragePath = dotCursorChats
		}
	default:
		return StoragePaths{}, fmt.Errorf("unsupported OS: %s (only macOS and Linux are supported)", runtime.GOOS)
	}

	return StoragePaths{
		WorkspaceStorage: filepath.Join(basePath, "workspaceStorage"),
		GlobalStorage:    filepath.Join(basePath, "globalStorage"),
		BasePath:         basePath,
		AgentStoragePath: agentStoragePath,
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

// HasAgentStorage checks if the agent storage directory exists
func (sp StoragePaths) HasAgentStorage() bool {
	if sp.AgentStoragePath == "" {
		return false
	}
	info, err := os.Stat(sp.AgentStoragePath)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// FindAgentStoreDBs scans the agent storage directory and returns a list of store.db file paths
func (sp StoragePaths) FindAgentStoreDBs() ([]string, error) {
	if !sp.HasAgentStorage() {
		return []string{}, nil
	}

	var storeDBs []string
	var dirsScanned int
	var dirsWithFiles int
	
	err := filepath.Walk(sp.AgentStoragePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Skip directories we can't access
			if info != nil && info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() {
			dirsScanned++
			// Check if this directory contains a store.db file
			storeDBPath := filepath.Join(path, "store.db")
			if _, err := os.Stat(storeDBPath); err == nil {
				dirsWithFiles++
			}
		}

		// Look for store.db files
		if !info.IsDir() && info.Name() == "store.db" {
			storeDBs = append(storeDBs, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to scan agent storage directory: %w", err)
	}

	// Log diagnostic information in verbose mode
	if len(storeDBs) == 0 && dirsScanned > 0 {
		LogInfo("Scanned %d directories in agent storage, found %d directories with files, but no store.db files", dirsScanned, dirsWithFiles)
	}

	return storeDBs, nil
}
