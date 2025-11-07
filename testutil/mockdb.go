package testutil

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

// CreateInMemoryDB creates an in-memory SQLite database for testing
func CreateInMemoryDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create in-memory database: %v", err)
	}

	// Create cursorDiskKV table
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS cursorDiskKV (
		key TEXT PRIMARY KEY,
		value TEXT
	)`
	if _, err := db.Exec(createTableSQL); err != nil {
		db.Close()
		t.Fatalf("Failed to create cursorDiskKV table: %v", err)
	}

	return db
}

// CreateTestDB creates a test database with sample data
func CreateTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db := CreateInMemoryDB(t)

	// Insert sample bubbles
	bubbles := []struct {
		key   string
		value string
	}{
		{
			key:   "bubbleId:chat1:bubble1",
			value: `{"bubbleId":"bubble1","chatId":"chat1","text":"Hello","timestamp":1000,"type":1}`,
		},
		{
			key:   "bubbleId:chat1:bubble2",
			value: `{"bubbleId":"bubble2","chatId":"chat1","text":"Hi there","timestamp":2000,"type":2}`,
		},
		{
			key:   "bubbleId:chat2:bubble3",
			value: `{"bubbleId":"bubble3","chatId":"chat2","text":"How are you?","timestamp":3000,"type":1}`,
		},
	}

	// Insert sample composers
	composers := []struct {
		key   string
		value string
	}{
		{
			key:   "composerData:composer1",
			value: `{"composerId":"composer1","name":"Test Conversation","createdAt":1000,"lastUpdatedAt":2000}`,
		},
		{
			key:   "composerData:composer2",
			value: `{"composerId":"composer2","name":"Another Conversation","createdAt":3000,"lastUpdatedAt":4000}`,
		},
	}

	// Insert sample message contexts
	contexts := []struct {
		key   string
		value string
	}{
		{
			key:   "messageRequestContext:composer1:context1",
			value: `{"bubbleId":"bubble1","composerId":"composer1","contextId":"context1"}`,
		},
	}

	// Insert all data
	insertSQL := "INSERT INTO cursorDiskKV (key, value) VALUES (?, ?)"
	stmt, err := db.Prepare(insertSQL)
	if err != nil {
		db.Close()
		t.Fatalf("Failed to prepare insert statement: %v", err)
	}
	defer stmt.Close()

	for _, bubble := range bubbles {
		if _, err := stmt.Exec(bubble.key, bubble.value); err != nil {
			db.Close()
			t.Fatalf("Failed to insert bubble: %v", err)
		}
	}

	for _, composer := range composers {
		if _, err := stmt.Exec(composer.key, composer.value); err != nil {
			db.Close()
			t.Fatalf("Failed to insert composer: %v", err)
		}
	}

	for _, ctx := range contexts {
		if _, err := stmt.Exec(ctx.key, ctx.value); err != nil {
			db.Close()
			t.Fatalf("Failed to insert context: %v", err)
		}
	}

	return db
}

// InsertBubble inserts a bubble into the database
func InsertBubble(t *testing.T, db *sql.DB, key, value string) {
	t.Helper()
	insertSQL := "INSERT INTO cursorDiskKV (key, value) VALUES (?, ?)"
	if _, err := db.Exec(insertSQL, key, value); err != nil {
		t.Fatalf("Failed to insert bubble: %v", err)
	}
}

// InsertComposer inserts a composer into the database
func InsertComposer(t *testing.T, db *sql.DB, key, value string) {
	t.Helper()
	insertSQL := "INSERT INTO cursorDiskKV (key, value) VALUES (?, ?)"
	if _, err := db.Exec(insertSQL, key, value); err != nil {
		t.Fatalf("Failed to insert composer: %v", err)
	}
}
