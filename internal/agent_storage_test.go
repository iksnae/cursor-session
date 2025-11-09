package internal

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/iksnae/cursor-session/testutil"
	_ "modernc.org/sqlite"
)

func TestQueryBlobsTable(t *testing.T) {
	db := testutil.CreateInMemoryDB(t)
	defer func() { _ = db.Close() }()

	// Create blobs table with key-value structure
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS blobs (
		key TEXT PRIMARY KEY,
		value TEXT
	)`
	if _, err := db.Exec(createTableSQL); err != nil {
		t.Fatalf("Failed to create blobs table: %v", err)
	}

	// Insert test data
	insertSQL := "INSERT INTO blobs (key, value) VALUES (?, ?)"
	testData := []struct {
		key   string
		value string
	}{
		{"bubble1", `{"bubbleId":"bubble1","chatId":"chat1","text":"Hello","timestamp":1000,"type":1}`},
		{"bubble2", `{"bubbleId":"bubble2","chatId":"chat1","text":"Hi there","timestamp":2000,"type":2}`},
	}

	for _, data := range testData {
		if _, err := db.Exec(insertSQL, data.key, data.value); err != nil {
			t.Fatalf("Failed to insert test data: %v", err)
		}
	}

	// Query blobs
	blobs, err := QueryBlobsTable(db)
	if err != nil {
		t.Fatalf("QueryBlobsTable() error = %v", err)
	}

	if len(blobs) != 2 {
		t.Errorf("QueryBlobsTable() returned %d blobs, want 2", len(blobs))
	}
}

func TestQueryBlobsTable_NoTable(t *testing.T) {
	db := testutil.CreateInMemoryDB(t)
	defer func() { _ = db.Close() }()

	// Don't create blobs table
	blobs, err := QueryBlobsTable(db)
	if err != nil {
		t.Fatalf("QueryBlobsTable() error = %v", err)
	}

	if len(blobs) != 0 {
		t.Errorf("QueryBlobsTable() returned %d blobs, want 0", len(blobs))
	}
}

func TestQueryMetaTable(t *testing.T) {
	db := testutil.CreateInMemoryDB(t)
	defer func() { _ = db.Close() }()

	// Create meta table
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS meta (
		key TEXT PRIMARY KEY,
		value TEXT
	)`
	if _, err := db.Exec(createTableSQL); err != nil {
		t.Fatalf("Failed to create meta table: %v", err)
	}

	// Insert test data
	insertSQL := "INSERT INTO meta (key, value) VALUES (?, ?)"
	testData := []struct {
		key   string
		value string
	}{
		{"context1", `{"contextId":"context1","composerId":"composer1","bubbleId":"bubble1"}`},
	}

	for _, data := range testData {
		if _, err := db.Exec(insertSQL, data.key, data.value); err != nil {
			t.Fatalf("Failed to insert test data: %v", err)
		}
	}

	// Query meta
	meta, err := QueryMetaTable(db)
	if err != nil {
		t.Fatalf("QueryMetaTable() error = %v", err)
	}

	if len(meta) != 1 {
		t.Errorf("QueryMetaTable() returned %d entries, want 1", len(meta))
	}
}

func TestQueryMetaTable_NoTable(t *testing.T) {
	db := testutil.CreateInMemoryDB(t)
	defer func() { _ = db.Close() }()

	// Don't create meta table
	meta, err := QueryMetaTable(db)
	if err != nil {
		t.Fatalf("QueryMetaTable() error = %v", err)
	}

	if len(meta) != 0 {
		t.Errorf("QueryMetaTable() returned %d entries, want 0", len(meta))
	}
}

func TestLoadSessionFromStoreDB(t *testing.T) {
	// Create a temporary database file
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "store.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Create blobs table
	createBlobsSQL := `
	CREATE TABLE IF NOT EXISTS blobs (
		key TEXT PRIMARY KEY,
		value TEXT
	)`
	if _, err := db.Exec(createBlobsSQL); err != nil {
		t.Fatalf("Failed to create blobs table: %v", err)
	}

	// Create meta table
	createMetaSQL := `
	CREATE TABLE IF NOT EXISTS meta (
		key TEXT PRIMARY KEY,
		value TEXT
	)`
	if _, err := db.Exec(createMetaSQL); err != nil {
		t.Fatalf("Failed to create meta table: %v", err)
	}

	// Insert test data
	insertBlobSQL := "INSERT INTO blobs (key, value) VALUES (?, ?)"
	bubbleData := `{"bubbleId":"bubble1","chatId":"chat1","text":"Hello","timestamp":1000,"type":1}`
	if _, err := db.Exec(insertBlobSQL, "bubble1", bubbleData); err != nil {
		t.Fatalf("Failed to insert bubble: %v", err)
	}

	composerData := `{"composerId":"composer1","name":"Test","createdAt":1000,"lastUpdatedAt":2000}`
	if _, err := db.Exec(insertBlobSQL, "composer1", composerData); err != nil {
		t.Fatalf("Failed to insert composer: %v", err)
	}

	insertMetaSQL := "INSERT INTO meta (key, value) VALUES (?, ?)"
	contextData := `{"contextId":"context1","composerId":"composer1","bubbleId":"bubble1"}`
	if _, err := db.Exec(insertMetaSQL, "context1", contextData); err != nil {
		t.Fatalf("Failed to insert context: %v", err)
	}

	_ = db.Close()

	// Load session
	bubbles, composers, contexts, err := LoadSessionFromStoreDB(dbPath)
	if err != nil {
		t.Fatalf("LoadSessionFromStoreDB() error = %v", err)
	}

	if len(bubbles) == 0 {
		t.Error("LoadSessionFromStoreDB() returned no bubbles")
	}

	if len(composers) == 0 {
		t.Error("LoadSessionFromStoreDB() returned no composers")
	}

	if len(contexts) == 0 {
		t.Error("LoadSessionFromStoreDB() returned no contexts")
	}
}

func TestLoadSessionFromStoreDB_Nonexistent(t *testing.T) {
	bubbles, composers, contexts, err := LoadSessionFromStoreDB("/nonexistent/path/store.db")
	if err == nil {
		t.Error("LoadSessionFromStoreDB() should return error for nonexistent file")
	}

	if bubbles != nil {
		t.Error("LoadSessionFromStoreDB() should return nil bubbles on error")
	}

	if composers != nil {
		t.Error("LoadSessionFromStoreDB() should return nil composers on error")
	}

	if contexts != nil {
		t.Error("LoadSessionFromStoreDB() should return nil contexts on error")
	}
}

func TestNewAgentStorageReader(t *testing.T) {
	paths := []string{"/path1/store.db", "/path2/store.db"}
	reader := NewAgentStorageReader(paths)

	if reader == nil {
		t.Fatal("NewAgentStorageReader() returned nil")
	}

	if len(reader.storeDBPaths) != 2 {
		t.Errorf("NewAgentStorageReader() stored %d paths, want 2", len(reader.storeDBPaths))
	}
}

func TestLoadAllSessionsFromAgentStorage(t *testing.T) {
	// Create temporary database files
	tmpDir := t.TempDir()
	dbPath1 := filepath.Join(tmpDir, "store1.db")
	dbPath2 := filepath.Join(tmpDir, "store2.db")

	// Create first database
	db1, err := sql.Open("sqlite", dbPath1)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	createBlobsSQL := `CREATE TABLE IF NOT EXISTS blobs (key TEXT PRIMARY KEY, value TEXT)`
	if _, err := db1.Exec(createBlobsSQL); err != nil {
		t.Fatalf("Failed to create blobs table: %v", err)
	}
	insertSQL := "INSERT INTO blobs (key, value) VALUES (?, ?)"
	if _, err := db1.Exec(insertSQL, "bubble1", `{"bubbleId":"bubble1","chatId":"chat1","text":"Hello","timestamp":1000,"type":1}`); err != nil {
		t.Fatalf("Failed to insert data: %v", err)
	}
	_ = db1.Close()

	// Create second database
	db2, err := sql.Open("sqlite", dbPath2)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	if _, err := db2.Exec(createBlobsSQL); err != nil {
		t.Fatalf("Failed to create blobs table: %v", err)
	}
	if _, err := db2.Exec(insertSQL, "bubble2", `{"bubbleId":"bubble2","chatId":"chat2","text":"Hi","timestamp":2000,"type":2}`); err != nil {
		t.Fatalf("Failed to insert data: %v", err)
	}
	_ = db2.Close()

	// Load all sessions
	reader := NewAgentStorageReader([]string{dbPath1, dbPath2})
	bubbles, _, _, err := reader.LoadAllSessionsFromAgentStorage()
	if err != nil {
		t.Fatalf("LoadAllSessionsFromAgentStorage() error = %v", err)
	}

	if len(bubbles) < 2 {
		t.Errorf("LoadAllSessionsFromAgentStorage() returned %d bubbles, want at least 2", len(bubbles))
	}
}

func TestLoadAllSessionsFromAgentStorage_Empty(t *testing.T) {
	reader := NewAgentStorageReader([]string{})
	bubbles, composers, contexts, err := reader.LoadAllSessionsFromAgentStorage()
	if err != nil {
		t.Fatalf("LoadAllSessionsFromAgentStorage() error = %v", err)
	}

	if len(bubbles) != 0 {
		t.Errorf("LoadAllSessionsFromAgentStorage() returned %d bubbles, want 0", len(bubbles))
	}

	if len(composers) != 0 {
		t.Errorf("LoadAllSessionsFromAgentStorage() returned %d composers, want 0", len(composers))
	}

	if len(contexts) != 0 {
		t.Errorf("LoadAllSessionsFromAgentStorage() returned %d contexts, want 0", len(contexts))
	}
}

func TestExtractSessionIDFromPath(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"/home/user/.cursor/chats/hash123/session-abc/store.db", "session-abc"},
		{"/path/to/session-id/store.db", "session-id"},
		{"/store.db", "/"},
	}

	for _, tt := range tests {
		result := extractSessionIDFromPath(tt.path)
		if result != tt.expected {
			t.Errorf("extractSessionIDFromPath(%q) = %q, want %q", tt.path, result, tt.expected)
		}
	}
}

func TestParseBubbleFromData(t *testing.T) {
	data := map[string]interface{}{
		"bubbleId":  "bubble1",
		"chatId":    "chat1",
		"text":      "Hello",
		"timestamp": float64(1000),
		"type":      float64(1),
	}

	bubble, err := parseBubbleFromData("key", data, "session1")
	if err != nil {
		t.Fatalf("parseBubbleFromData() error = %v", err)
	}

	if bubble.BubbleID != "bubble1" {
		t.Errorf("parseBubbleFromData() BubbleID = %q, want %q", bubble.BubbleID, "bubble1")
	}

	if bubble.ChatID != "chat1" {
		t.Errorf("parseBubbleFromData() ChatID = %q, want %q", bubble.ChatID, "chat1")
	}

	if bubble.Text != "Hello" {
		t.Errorf("parseBubbleFromData() Text = %q, want %q", bubble.Text, "Hello")
	}
}

func TestParseComposerFromData(t *testing.T) {
	data := map[string]interface{}{
		"composerId":    "composer1",
		"name":          "Test Conversation",
		"createdAt":     float64(1000),
		"lastUpdatedAt": float64(2000),
	}

	composer, err := parseComposerFromData("key", data)
	if err != nil {
		t.Fatalf("parseComposerFromData() error = %v", err)
	}

	if composer.ComposerID != "composer1" {
		t.Errorf("parseComposerFromData() ComposerID = %q, want %q", composer.ComposerID, "composer1")
	}

	if composer.Name != "Test Conversation" {
		t.Errorf("parseComposerFromData() Name = %q, want %q", composer.Name, "Test Conversation")
	}
}

func TestParseContextFromData(t *testing.T) {
	data := map[string]interface{}{
		"contextId":  "context1",
		"bubbleId":   "bubble1",
		"composerId": "composer1",
	}

	context, err := parseContextFromData("key", data)
	if err != nil {
		t.Fatalf("parseContextFromData() error = %v", err)
	}

	if context.ContextID != "context1" {
		t.Errorf("parseContextFromData() ContextID = %q, want %q", context.ContextID, "context1")
	}

	if context.BubbleID != "bubble1" {
		t.Errorf("parseContextFromData() BubbleID = %q, want %q", context.BubbleID, "bubble1")
	}

	if context.ComposerID != "composer1" {
		t.Errorf("parseContextFromData() ComposerID = %q, want %q", context.ComposerID, "composer1")
	}
}
