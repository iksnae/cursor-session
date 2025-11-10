package internal

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	_ "modernc.org/sqlite"
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
	return GetStoragePaths("")
}

// GetStoragePaths returns storage paths, using customPath if provided, otherwise auto-detecting
// customPath can be:
//   - Empty string: auto-detect default paths
//   - Path to a database file (state.vscdb or store.db): use that file
//   - Path to globalStorage directory: use that directory
//   - Path to agent storage directory: use that directory
func GetStoragePaths(customPath string) (StoragePaths, error) {
	// If no custom path provided, use auto-detection
	if customPath == "" {
		return detectStoragePathsAuto()
	}

	// Check if custom path exists
	info, err := os.Stat(customPath)
	if err != nil {
		return StoragePaths{}, fmt.Errorf("custom storage path does not exist: %w", err)
	}

	// If it's a file, determine what type of database it is
	if !info.IsDir() {
		filename := filepath.Base(customPath)
		dir := filepath.Dir(customPath)

		// Check if it's state.vscdb (globalStorage format)
		if filename == "state.vscdb" {
			// Treat parent directory as globalStorage
			return StoragePaths{
				GlobalStorage:    dir,
				BasePath:         filepath.Dir(dir),
				WorkspaceStorage: filepath.Join(filepath.Dir(dir), "workspaceStorage"),
				AgentStoragePath: "",
			}, nil
		}

		// Check if it's store.db (agent storage format)
		if filename == "store.db" {
			// For agent storage, the store.db is typically in a subdirectory like {hash}/{session-id}/store.db
			// We'll use the directory containing the store.db as the agent storage root
			// This allows FindAgentStoreDBs() to find this specific file and any others in the directory tree
			agentRoot := dir

			// Try to find a reasonable root by walking up a few levels
			// This handles cases where the file is deep in a nested structure
			for i := 0; i < 3; i++ {
				parent := filepath.Dir(agentRoot)
				if parent == agentRoot {
					break
				}
				agentRoot = parent
			}

			home, _ := os.UserHomeDir()
			basePath := filepath.Join(home, ".config/Cursor/User")
			if runtime.GOOS == "darwin" {
				basePath = filepath.Join(home, "Library/Application Support/Cursor/User")
			}

			return StoragePaths{
				GlobalStorage:    filepath.Join(basePath, "globalStorage"),
				BasePath:         basePath,
				WorkspaceStorage: filepath.Join(basePath, "workspaceStorage"),
				AgentStoragePath: agentRoot,
			}, nil
		}

		// Unknown file type
		return StoragePaths{}, fmt.Errorf("unsupported database file: %s (expected state.vscdb or store.db)", filename)
	}

	// It's a directory - check what type of storage directory it is
	// Check if it's a globalStorage directory (contains state.vscdb)
	stateVscdbPath := filepath.Join(customPath, "state.vscdb")
	if _, err := os.Stat(stateVscdbPath); err == nil {
		// It's a globalStorage directory
		return StoragePaths{
			GlobalStorage:    customPath,
			BasePath:         filepath.Dir(customPath),
			WorkspaceStorage: filepath.Join(filepath.Dir(customPath), "workspaceStorage"),
			AgentStoragePath: "",
		}, nil
	}

	// Check if it's an agent storage directory (contains store.db files in subdirectories)
	// We'll check by looking for at least one store.db file
	hasStoreDB := false
	err = filepath.Walk(customPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && info.Name() == "store.db" {
			hasStoreDB = true
			return filepath.SkipAll // Found one, that's enough
		}
		return nil
	})

	if err == nil && hasStoreDB {
		// It's an agent storage directory
		home, _ := os.UserHomeDir()
		basePath := filepath.Join(home, ".config/Cursor/User")
		if runtime.GOOS == "darwin" {
			basePath = filepath.Join(home, "Library/Application Support/Cursor/User")
		}

		return StoragePaths{
			GlobalStorage:    filepath.Join(basePath, "globalStorage"),
			BasePath:         basePath,
			WorkspaceStorage: filepath.Join(basePath, "workspaceStorage"),
			AgentStoragePath: customPath,
		}, nil
	}

	// Unknown directory type
	return StoragePaths{}, fmt.Errorf("directory does not appear to be a valid Cursor storage location (expected globalStorage directory with state.vscdb, or agent storage directory with store.db files)")
}

// detectStoragePathsAuto detects the Cursor storage paths based on the operating system
func detectStoragePathsAuto() (StoragePaths, error) {
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

	storeDBs := make([]string, 0) // Initialize as empty slice, not nil
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
		// Return empty slice instead of nil on error to maintain consistent return type
		return []string{}, fmt.Errorf("failed to scan agent storage directory: %w", err)
	}

	// Log diagnostic information in verbose mode
	if len(storeDBs) == 0 && dirsScanned > 0 {
		LogInfo("Scanned %d directories in agent storage, found %d directories with files, but no store.db files", dirsScanned, dirsWithFiles)
	}

	return storeDBs, nil
}

// CopyStoragePaths copies database files to a temporary location and returns updated paths
// along with a cleanup function. This helps avoid database locking issues when Cursor IDE is running.
// Returns:
//   - Updated StoragePaths pointing to copied files
//   - Cleanup function to remove temporary files (call when done)
//   - Error if copying fails
func CopyStoragePaths(paths StoragePaths) (StoragePaths, func() error, error) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "cursor-session-*")
	if err != nil {
		return StoragePaths{}, nil, fmt.Errorf("failed to create temporary directory: %w", err)
	}

	cleanup := func() error {
		return os.RemoveAll(tmpDir)
	}

	newPaths := paths

	// Copy globalStorage database if it exists
	if paths.GlobalStorageExists() {
		sourceDB := paths.GetGlobalStorageDBPath()
		destDB := filepath.Join(tmpDir, "state.vscdb")

		if err := copyDatabaseWithWAL(sourceDB, destDB); err != nil {
			_ = cleanup()
			return StoragePaths{}, nil, fmt.Errorf("failed to copy globalStorage database: %w", err)
		}

		LogInfo("Copied globalStorage database to temporary location: %s", destDB)
		newPaths.GlobalStorage = tmpDir
	}

	// Copy agent storage databases if they exist
	if paths.HasAgentStorage() {
		storeDBs, err := paths.FindAgentStoreDBs()
		if err != nil {
			_ = cleanup()
			return StoragePaths{}, nil, fmt.Errorf("failed to find agent storage databases: %w", err)
		}

		if len(storeDBs) > 0 {
			// Create agent storage directory structure in temp, preserving the original structure
			agentTmpDir := filepath.Join(tmpDir, "agent-storage")
			if err := os.MkdirAll(agentTmpDir, 0755); err != nil {
				_ = cleanup()
				return StoragePaths{}, nil, fmt.Errorf("failed to create agent storage temp directory: %w", err)
			}

			// Copy each store.db file, preserving the relative path structure
			// This ensures FindAgentStoreDBs() can find them with the same structure
			for i, sourceDB := range storeDBs {
				// Get relative path from agent storage root
				relPath, err := filepath.Rel(paths.AgentStoragePath, sourceDB)
				if err != nil {
					// If we can't get relative path, create a simple structure
					relPath = fmt.Sprintf("session_%d/store.db", i)
				}

				// Build destination path preserving structure
				destDB := filepath.Join(agentTmpDir, relPath)

				// Ensure parent directory exists
				if err := os.MkdirAll(filepath.Dir(destDB), 0755); err != nil {
					_ = cleanup()
					return StoragePaths{}, nil, fmt.Errorf("failed to create directory for copied database: %w", err)
				}

				if err := copyDatabaseWithWAL(sourceDB, destDB); err != nil {
					_ = cleanup()
					return StoragePaths{}, nil, fmt.Errorf("failed to copy agent storage database %s: %w", sourceDB, err)
				}

				LogInfo("Copied agent storage database %d/%d: %s", i+1, len(storeDBs), destDB)
			}

			// Update paths to point to copied files
			// FindAgentStoreDBs() will now scan the copied directory structure
			newPaths.AgentStoragePath = agentTmpDir
			LogInfo("Copied %d agent storage database(s) to temporary location", len(storeDBs))
		}
	}

	return newPaths, cleanup, nil
}

// copyFile copies a file from source to destination
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer func() {
		_ = sourceFile.Close()
	}()

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer func() {
		_ = destFile.Close()
	}()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	// Sync to ensure data is written to disk
	if err := destFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync destination file: %w", err)
	}

	return nil
}

// copyDatabaseWithWAL copies a database file along with its associated WAL and SHM files if they exist.
// After copying, it checkpoints the WAL file to merge it into the main database, ensuring a consistent
// and complete copy. This is important because SQLite in WAL mode stores recent transactions in the WAL file.
func copyDatabaseWithWAL(srcDB, dstDB string) error {
	// Copy the main database file
	if err := copyFile(srcDB, dstDB); err != nil {
		return err
	}

	// Check for and copy WAL file if it exists
	srcWAL := srcDB + "-wal"
	dstWAL := dstDB + "-wal"
	hasWAL := false
	if _, err := os.Stat(srcWAL); err == nil {
		if err := copyFile(srcWAL, dstWAL); err != nil {
			// Log warning but don't fail - WAL file copy is best effort
			LogWarn("Failed to copy WAL file %s: %v", srcWAL, err)
		} else {
			LogInfo("Copied WAL file: %s", dstWAL)
			hasWAL = true
		}
	}

	// Check for and copy SHM file if it exists
	srcSHM := srcDB + "-shm"
	dstSHM := dstDB + "-shm"
	if _, err := os.Stat(srcSHM); err == nil {
		if err := copyFile(srcSHM, dstSHM); err != nil {
			// Log warning but don't fail - SHM file copy is best effort
			LogWarn("Failed to copy SHM file %s: %v", srcSHM, err)
		} else {
			LogInfo("Copied SHM file: %s", dstSHM)
		}
	}

	// If we copied a WAL file, checkpoint it to merge into the main database
	// This ensures the copied database is complete and consistent
	if hasWAL {
		if err := checkpointWAL(dstDB); err != nil {
			// Log warning but don't fail - checkpoint is best effort
			// The database should still be readable, just might be missing recent WAL transactions
			LogWarn("Failed to checkpoint WAL for copied database %s: %v (database may be incomplete)", dstDB, err)
		} else {
			LogInfo("Checkpointed WAL file for copied database: %s", dstDB)
		}
	}

	return nil
}

// checkpointWAL checkpoints a SQLite database's WAL file, merging it into the main database.
// This is necessary when copying databases in WAL mode to ensure all data is in the main file.
func checkpointWAL(dbPath string) error {
	// Open the copied database in read-write mode to checkpoint
	// Note: We use mode=rwc to create if needed, but the file should already exist
	db, err := sql.Open("sqlite", dbPath+"?mode=rwc")
	if err != nil {
		return fmt.Errorf("failed to open database for checkpoint: %w", err)
	}
	defer func() { _ = db.Close() }()

	// Test connection
	if err := db.Ping(); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	// Run PRAGMA wal_checkpoint(TRUNCATE) to checkpoint and truncate the WAL file
	// This merges all WAL transactions into the main database and removes the WAL file
	// Note: PRAGMA wal_checkpoint returns three integers (checkpointed, pages_written, pages_checkpointed)
	// but the modernc.org/sqlite driver may not support returning values from PRAGMA statements.
	// We'll execute it and check for errors.
	_, err = db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
	if err != nil {
		return fmt.Errorf("failed to checkpoint WAL: %w", err)
	}

	// The checkpoint should have merged the WAL into the main database
	// The WAL file will be truncated (emptied) but may still exist
	LogInfo("WAL checkpoint completed for %s", dbPath)

	return nil
}
