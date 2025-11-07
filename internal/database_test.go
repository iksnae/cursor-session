package internal

import (
	"path/filepath"
	"testing"

	"github.com/iksnae/cursor-session/testutil"
)

func TestOpenDatabase(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) string
		wantErr bool
	}{
		{
			name: "valid database",
			setup: func(t *testing.T) string {
				tmpDir := testutil.CreateTempDir(t)
				dbPath := filepath.Join(tmpDir, "test.db")
				// Create a real file database
				testutil.CreateSQLiteFixture(t, dbPath)
				return dbPath
			},
			wantErr: false,
		},
		{
			name: "non-existent database",
			setup: func(t *testing.T) string {
				tmpDir := testutil.CreateTempDir(t)
				dbPath := filepath.Join(tmpDir, "nonexistent.db")
				// SQLite in read-only mode should fail on non-existent file
				// The error typically comes from Ping, not Open
				return dbPath
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbPath := tt.setup(t)
			db, err := OpenDatabase(dbPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("OpenDatabase() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if db == nil {
					t.Error("OpenDatabase() returned nil database")
					return
				}
				// Test ping
				if err := db.Ping(); err != nil {
					t.Errorf("Database ping failed: %v", err)
				}
				db.Close()
			}
		})
	}
}

func TestQueryCursorDiskKV(t *testing.T) {
	db := testutil.CreateTestDB(t)
	defer db.Close()

	tests := []struct {
		name    string
		pattern string
		want    int // expected number of results
		wantErr bool
	}{
		{
			name:    "query bubbles",
			pattern: "bubbleId:%",
			want:    3,
			wantErr: false,
		},
		{
			name:    "query composers",
			pattern: "composerData:%",
			want:    2,
			wantErr: false,
		},
		{
			name:    "query message contexts",
			pattern: "messageRequestContext:%",
			want:    1,
			wantErr: false,
		},
		{
			name:    "query non-existent pattern",
			pattern: "nonexistent:%",
			want:    0,
			wantErr: false,
		},
		{
			name:    "query specific chat",
			pattern: "bubbleId:chat1:%",
			want:    2,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pairs, err := QueryCursorDiskKV(db, tt.pattern)
			if (err != nil) != tt.wantErr {
				t.Errorf("QueryCursorDiskKV() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(pairs) != tt.want {
					t.Errorf("QueryCursorDiskKV() returned %d pairs, want %d", len(pairs), tt.want)
				}

				// Verify all pairs have keys and values
				for i, pair := range pairs {
					if pair.Key == "" {
						t.Errorf("Pair %d has empty key", i)
					}
					if pair.Value == "" {
						t.Errorf("Pair %d has empty value", i)
					}
					// Verify key matches pattern
					if !matchesPattern(pair.Key, tt.pattern) {
						t.Errorf("Pair %d key %q does not match pattern %q", i, pair.Key, tt.pattern)
					}
				}
			}
		})
	}
}

func TestQueryCursorDiskKV_NullValues(t *testing.T) {
	db := testutil.CreateInMemoryDB(t)
	defer db.Close()

	// Insert a row with NULL value
	_, err := db.Exec("INSERT INTO cursorDiskKV (key, value) VALUES (?, ?)", "test:key1", nil)
	if err != nil {
		t.Fatalf("Failed to insert NULL value: %v", err)
	}

	// Insert a row with valid value
	_, err = db.Exec("INSERT INTO cursorDiskKV (key, value) VALUES (?, ?)", "test:key2", "value2")
	if err != nil {
		t.Fatalf("Failed to insert valid value: %v", err)
	}

	pairs, err := QueryCursorDiskKV(db, "test:%")
	if err != nil {
		t.Fatalf("QueryCursorDiskKV() error = %v", err)
	}

	// Should only return the non-NULL value
	if len(pairs) != 1 {
		t.Errorf("QueryCursorDiskKV() returned %d pairs, want 1", len(pairs))
	}

	if pairs[0].Key != "test:key2" {
		t.Errorf("QueryCursorDiskKV() returned key %q, want test:key2", pairs[0].Key)
	}
}

// matchesPattern checks if a key matches a LIKE pattern (simplified)
func matchesPattern(key, pattern string) bool {
	// Convert LIKE pattern to simple prefix/suffix check
	if pattern[len(pattern)-1] == '%' {
		prefix := pattern[:len(pattern)-1]
		return len(key) >= len(prefix) && key[:len(prefix)] == prefix
	}
	return key == pattern
}


