package testutil

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// CreateSQLiteFixture creates a SQLite database fixture with sample data
func CreateSQLiteFixture(t *testing.T, dbPath string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		t.Fatalf("Failed to create fixture directory: %v", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Create cursorDiskKV table
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS cursorDiskKV (
		key TEXT PRIMARY KEY,
		value TEXT
	)`
	if _, err := db.Exec(createTableSQL); err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Insert sample data
	bubbleData := map[string]interface{}{
		"bubbleId":  "bubble1",
		"chatId":    "chat1",
		"text":      "Hello world",
		"timestamp": time.Now().UnixMilli(),
		"type":      1,
	}
	bubbleJSON, _ := json.Marshal(bubbleData)

	composerData := map[string]interface{}{
		"composerId":    "composer1",
		"name":          "Test Conversation",
		"createdAt":     time.Now().UnixMilli(),
		"lastUpdatedAt": time.Now().UnixMilli(),
	}
	composerJSON, _ := json.Marshal(composerData)

	insertSQL := "INSERT INTO cursorDiskKV (key, value) VALUES (?, ?)"
	if _, err := db.Exec(insertSQL, "bubbleId:chat1:bubble1", string(bubbleJSON)); err != nil {
		t.Fatalf("Failed to insert bubble: %v", err)
	}
	if _, err := db.Exec(insertSQL, "composerData:composer1", string(composerJSON)); err != nil {
		t.Fatalf("Failed to insert composer: %v", err)
	}
}

// CreateCacheFixture creates a cache file fixture
func CreateCacheFixture(t *testing.T, cachePath string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(cachePath), 0755); err != nil {
		t.Fatalf("Failed to create cache directory: %v", err)
	}
	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		t.Fatalf("Failed to write cache file: %v", err)
	}
}

// CreateWorkspaceFixture creates a workspace fixture structure
func CreateWorkspaceFixture(t *testing.T, basePath string, workspaceHash string) string {
	t.Helper()
	workspaceDir := filepath.Join(basePath, "workspaceStorage", workspaceHash)
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("Failed to create workspace directory: %v", err)
	}

	// Create workspace.json
	workspaceJSON := map[string]interface{}{
		"folder": "/path/to/workspace",
	}
	jsonData, _ := json.Marshal(workspaceJSON)
	workspaceJSONPath := filepath.Join(workspaceDir, "workspace.json")
	if err := os.WriteFile(workspaceJSONPath, jsonData, 0644); err != nil {
		t.Fatalf("Failed to write workspace.json: %v", err)
	}

	return workspaceDir
}

// CreateMockCursorDir creates a mock Cursor directory structure
func CreateMockCursorDir(t *testing.T) string {
	t.Helper()
	tmpDir := CreateTempDir(t)

	// Create workspaceStorage structure
	workspaceHash := "workspace-hash-123"
	workspaceDir := CreateWorkspaceFixture(t, tmpDir, workspaceHash)

	// Create SQLite database in workspace
	dbPath := filepath.Join(workspaceDir, "state.vscdb")
	CreateSQLiteFixture(t, dbPath)

	// Create WebStorage structure
	webDir := filepath.Join(tmpDir, "WebStorage", "1", "CacheStorage", "cache-name")
	if err := os.MkdirAll(webDir, 0755); err != nil {
		t.Fatalf("Failed to create WebStorage directory: %v", err)
	}

	// Create mock cache file
	cachePath := filepath.Join(webDir, "data_0")
	cacheData := []byte(`{"messages":[{"role":"user","content":"test"}]}`)
	CreateCacheFixture(t, cachePath, cacheData)

	return tmpDir
}
