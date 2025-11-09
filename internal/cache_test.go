package internal

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/iksnae/cursor-session/testutil"
)

func TestNewCacheManager(t *testing.T) {
	cacheDir := testutil.CreateTempDir(t)
	cm := NewCacheManager(cacheDir)
	if cm == nil {
		t.Fatal("NewCacheManager() returned nil")
	}
	if cm.cacheDir != cacheDir {
		t.Errorf("NewCacheManager() cacheDir = %q, want %q", cm.cacheDir, cacheDir)
	}
}

func TestCacheManager_EnsureCacheDir(t *testing.T) {
	cacheDir := testutil.CreateTempDir(t)
	cm := NewCacheManager(cacheDir)

	err := cm.EnsureCacheDir()
	if err != nil {
		t.Errorf("EnsureCacheDir() error = %v", err)
	}

	// Verify directory exists
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		t.Error("Cache directory was not created")
	}
}

func TestCacheManager_GetIndexPath(t *testing.T) {
	cacheDir := testutil.CreateTempDir(t)
	cm := NewCacheManager(cacheDir)

	expected := filepath.Join(cacheDir, "sessions.yaml")
	if got := cm.GetIndexPath(); got != expected {
		t.Errorf("GetIndexPath() = %q, want %q", got, expected)
	}
}

func TestCacheManager_GetSessionPath(t *testing.T) {
	cacheDir := testutil.CreateTempDir(t)
	cm := NewCacheManager(cacheDir)

	sessionID := "test-session-123"
	expected := filepath.Join(cacheDir, "session_test-session-123.json")
	if got := cm.GetSessionPath(sessionID); got != expected {
		t.Errorf("GetSessionPath() = %q, want %q", got, expected)
	}
}

func TestCacheManager_IsCacheValid(t *testing.T) {
	cacheDir := testutil.CreateTempDir(t)
	cm := NewCacheManager(cacheDir)
	if err := cm.EnsureCacheDir(); err != nil {
		t.Fatalf("EnsureCacheDir() error = %v", err)
	}

	dbPath := filepath.Join(cacheDir, "test.db")
	// Create a simple test database file
	createTestDBFile(t, dbPath)

	tests := []struct {
		name    string
		setup   func()
		want    bool
		wantErr bool
	}{
		{
			name: "cache does not exist",
			setup: func() {
				// Don't create cache
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "cache exists and is valid",
			setup: func() {
				index := &SessionIndex{
					Metadata: CacheMetadata{
						DatabasePath:    dbPath,
						DatabaseModTime: getFileModTime(t, dbPath),
						CacheVersion:    "1.0",
					},
				}
				if err := cm.SaveIndex(index); err != nil {
					t.Fatalf("SaveIndex() error = %v", err)
				}
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "cache exists but database path mismatch",
			setup: func() {
				index := &SessionIndex{
					Metadata: CacheMetadata{
						DatabasePath:    "/different/path.db",
						DatabaseModTime: time.Now(),
						CacheVersion:    "1.0",
					},
				}
				if err := cm.SaveIndex(index); err != nil {
					t.Fatalf("SaveIndex() error = %v", err)
				}
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "cache exists but database modified",
			setup: func() {
				index := &SessionIndex{
					Metadata: CacheMetadata{
						DatabasePath:    dbPath,
						DatabaseModTime: time.Now().Add(-time.Hour),
						CacheVersion:    "1.0",
					},
				}
				if err := cm.SaveIndex(index); err != nil {
					t.Fatalf("SaveIndex() error = %v", err)
				}
			},
			want:    false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up previous cache
			_ = os.Remove(cm.GetIndexPath())
			tt.setup()

			got, err := cm.IsCacheValid(dbPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("IsCacheValid() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("IsCacheValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCacheManager_SaveAndLoadIndex(t *testing.T) {
	cacheDir := testutil.CreateTempDir(t)
	cm := NewCacheManager(cacheDir)
	if err := cm.EnsureCacheDir(); err != nil {
		t.Fatalf("EnsureCacheDir() error = %v", err)
	}

	index := &SessionIndex{
		Sessions: []SessionIndexEntry{
			{
				ID:           "session1",
				ComposerID:   "composer1",
				Name:         "Test Session",
				MessageCount: 5,
			},
		},
		Metadata: CacheMetadata{
			DatabasePath:    "/test/path.db",
			DatabaseModTime: time.Now(),
			CacheVersion:    "1.0",
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
	}

	err := cm.SaveIndex(index)
	if err != nil {
		t.Fatalf("SaveIndex() error = %v", err)
	}

	loaded, err := cm.LoadIndex()
	if err != nil {
		t.Fatalf("LoadIndex() error = %v", err)
	}

	if len(loaded.Sessions) != len(index.Sessions) {
		t.Errorf("LoadIndex() returned %d sessions, want %d", len(loaded.Sessions), len(index.Sessions))
	}

	if loaded.Sessions[0].ID != index.Sessions[0].ID {
		t.Errorf("LoadIndex() session ID = %q, want %q", loaded.Sessions[0].ID, index.Sessions[0].ID)
	}
}

func TestCacheManager_SaveAndLoadSession(t *testing.T) {
	cacheDir := testutil.CreateTempDir(t)
	cm := NewCacheManager(cacheDir)
	if err := cm.EnsureCacheDir(); err != nil {
		t.Fatalf("EnsureCacheDir() error = %v", err)
	}

	session := CreateTestSession("test-session")

	err := cm.SaveSession(session)
	if err != nil {
		t.Fatalf("SaveSession() error = %v", err)
	}

	loaded, err := cm.LoadSession(session.ID)
	if err != nil {
		t.Fatalf("LoadSession() error = %v", err)
	}

	if loaded.ID != session.ID {
		t.Errorf("LoadSession() ID = %q, want %q", loaded.ID, session.ID)
	}

	if len(loaded.Messages) != len(session.Messages) {
		t.Errorf("LoadSession() returned %d messages, want %d", len(loaded.Messages), len(session.Messages))
	}
}

func TestCacheManager_LoadAllSessions(t *testing.T) {
	cacheDir := testutil.CreateTempDir(t)
	cm := NewCacheManager(cacheDir)
	if err := cm.EnsureCacheDir(); err != nil {
		t.Fatalf("EnsureCacheDir() error = %v", err)
	}

	session1 := CreateTestSession("session1")
	session2 := CreateTestSession("session2")

	if err := cm.SaveSession(session1); err != nil {
		t.Fatalf("SaveSession() error = %v", err)
	}
	if err := cm.SaveSession(session2); err != nil {
		t.Fatalf("SaveSession() error = %v", err)
	}

	// Create index
	index := &SessionIndex{
		Sessions: []SessionIndexEntry{
			{ID: session1.ID},
			{ID: session2.ID},
		},
		Metadata: CacheMetadata{
			CacheVersion: "1.0",
		},
	}
	if err := cm.SaveIndex(index); err != nil {
		t.Fatalf("SaveIndex() error = %v", err)
	}

	sessions, err := cm.LoadAllSessions()
	if err != nil {
		t.Fatalf("LoadAllSessions() error = %v", err)
	}

	if len(sessions) != 2 {
		t.Errorf("LoadAllSessions() returned %d sessions, want 2", len(sessions))
	}
}

func TestCacheManager_GetCacheDir(t *testing.T) {
	cacheDir := testutil.CreateTempDir(t)
	cm := NewCacheManager(cacheDir)

	if got := cm.GetCacheDir(); got != cacheDir {
		t.Errorf("GetCacheDir() = %q, want %q", got, cacheDir)
	}
}

func getFileModTime(t *testing.T, path string) time.Time {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}
	return info.ModTime()
}

func createTestDBFile(t *testing.T, dbPath string) {
	t.Helper()
	testutil.CreateSQLiteFixture(t, dbPath)
}

func TestCacheManager_SaveSessionAndUpdateIndex(t *testing.T) {
	cacheDir := testutil.CreateTempDir(t)
	cm := NewCacheManager(cacheDir)
	if err := cm.EnsureCacheDir(); err != nil {
		t.Fatalf("EnsureCacheDir() error = %v", err)
	}

	dbPath := filepath.Join(cacheDir, "test.db")
	createTestDBFile(t, dbPath)

	session := CreateTestSession("test-session")
	session.Metadata.ComposerID = "composer-123"

	err := cm.SaveSessionAndUpdateIndex(session, dbPath)
	if err != nil {
		t.Fatalf("SaveSessionAndUpdateIndex() error = %v", err)
	}

	// Verify session was saved
	loaded, err := cm.LoadSession(session.ID)
	if err != nil {
		t.Fatalf("LoadSession() error = %v", err)
	}
	if loaded.ID != session.ID {
		t.Errorf("LoadSession() ID = %q, want %q", loaded.ID, session.ID)
	}

	// Verify index was updated
	index, err := cm.LoadIndex()
	if err != nil {
		t.Fatalf("LoadIndex() error = %v", err)
	}
	if len(index.Sessions) != 1 {
		t.Errorf("LoadIndex() returned %d sessions, want 1", len(index.Sessions))
	}
	if index.Sessions[0].ComposerID != "composer-123" {
		t.Errorf("LoadIndex() ComposerID = %q, want 'composer-123'", index.Sessions[0].ComposerID)
	}

	// Test updating existing session
	session.Metadata.Name = "Updated Name"
	err = cm.SaveSessionAndUpdateIndex(session, dbPath)
	if err != nil {
		t.Fatalf("SaveSessionAndUpdateIndex() error = %v", err)
	}

	index, err = cm.LoadIndex()
	if err != nil {
		t.Fatalf("LoadIndex() error = %v", err)
	}
	if len(index.Sessions) != 1 {
		t.Errorf("LoadIndex() returned %d sessions after update, want 1", len(index.Sessions))
	}
	if index.Sessions[0].Name != "Updated Name" {
		t.Errorf("LoadIndex() Name = %q, want 'Updated Name'", index.Sessions[0].Name)
	}
}

func TestCacheManager_SaveSessions(t *testing.T) {
	cacheDir := testutil.CreateTempDir(t)
	cm := NewCacheManager(cacheDir)
	if err := cm.EnsureCacheDir(); err != nil {
		t.Fatalf("EnsureCacheDir() error = %v", err)
	}

	dbPath := filepath.Join(cacheDir, "test.db")
	createTestDBFile(t, dbPath)

	session1 := CreateTestSession("session1")
	session1.Metadata.ComposerID = "composer1"
	session2 := CreateTestSession("session2")
	session2.Metadata.ComposerID = "composer2"

	sessions := []*Session{session1, session2}

	err := cm.SaveSessions(sessions, dbPath)
	if err != nil {
		t.Fatalf("SaveSessions() error = %v", err)
	}

	// Verify index was created
	index, err := cm.LoadIndex()
	if err != nil {
		t.Fatalf("LoadIndex() error = %v", err)
	}
	if len(index.Sessions) != 2 {
		t.Errorf("LoadIndex() returned %d sessions, want 2", len(index.Sessions))
	}

	// Verify sessions were saved
	loaded1, err := cm.LoadSession(session1.ID)
	if err != nil {
		t.Fatalf("LoadSession() error = %v", err)
	}
	if loaded1.ID != session1.ID {
		t.Errorf("LoadSession() ID = %q, want %q", loaded1.ID, session1.ID)
	}
}

func TestCacheManager_LoadConversations(t *testing.T) {
	cacheDir := testutil.CreateTempDir(t)
	cm := NewCacheManager(cacheDir)
	if err := cm.EnsureCacheDir(); err != nil {
		t.Fatalf("EnsureCacheDir() error = %v", err)
	}

	session := CreateTestSession("test-session")
	session.Metadata.ComposerID = "composer-123"
	session.Metadata.CreatedAt = "2024-01-01T00:00:00Z"
	session.Metadata.UpdatedAt = "2024-01-01T00:00:00Z"
	session.Messages[0].Timestamp = "2024-01-01T00:00:00Z"

	if err := cm.SaveSession(session); err != nil {
		t.Fatalf("SaveSession() error = %v", err)
	}

	// Create index
	index := &SessionIndex{
		Sessions: []SessionIndexEntry{
			{ID: session.ID},
		},
		Metadata: CacheMetadata{
			CacheVersion: "1.0",
		},
	}
	if err := cm.SaveIndex(index); err != nil {
		t.Fatalf("SaveIndex() error = %v", err)
	}

	conversations, err := cm.LoadConversations()
	if err != nil {
		t.Fatalf("LoadConversations() error = %v", err)
	}

	if len(conversations) != 1 {
		t.Errorf("LoadConversations() returned %d conversations, want 1", len(conversations))
	}

	if conversations[0].ComposerID != "composer-123" {
		t.Errorf("LoadConversations() ComposerID = %q, want 'composer-123'", conversations[0].ComposerID)
	}
}

func TestCacheManager_ClearCache(t *testing.T) {
	cacheDir := testutil.CreateTempDir(t)
	cm := NewCacheManager(cacheDir)
	if err := cm.EnsureCacheDir(); err != nil {
		t.Fatalf("EnsureCacheDir() error = %v", err)
	}

	// Create a session and index
	session := CreateTestSession("test-session")
	if err := cm.SaveSession(session); err != nil {
		t.Fatalf("SaveSession() error = %v", err)
	}

	index := &SessionIndex{
		Sessions: []SessionIndexEntry{
			{ID: session.ID},
		},
		Metadata: CacheMetadata{
			CacheVersion: "1.0",
		},
	}
	if err := cm.SaveIndex(index); err != nil {
		t.Fatalf("SaveIndex() error = %v", err)
	}

	// Clear cache
	err := cm.ClearCache()
	if err != nil {
		t.Fatalf("ClearCache() error = %v", err)
	}

	// Verify index is gone
	_, err = cm.LoadIndex()
	if err == nil {
		t.Error("LoadIndex() should fail after ClearCache()")
	}

	// Verify session file is gone
	_, err = cm.LoadSession(session.ID)
	if err == nil {
		t.Error("LoadSession() should fail after ClearCache()")
	}
}

func TestParseTimestamp(t *testing.T) {
	tests := []struct {
		name     string
		ts       string
		wantZero bool
	}{
		{
			name:     "valid RFC3339 timestamp",
			ts:       "2024-01-01T00:00:00Z",
			wantZero: false,
		},
		{
			name:     "empty string",
			ts:       "",
			wantZero: true,
		},
		{
			name:     "invalid format",
			ts:       "not-a-timestamp",
			wantZero: true,
		},
		{
			name:     "valid timestamp with milliseconds",
			ts:       "2024-01-01T12:34:56.789Z",
			wantZero: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseTimestamp(tt.ts)
			if tt.wantZero && result != 0 {
				t.Errorf("parseTimestamp(%q) = %d, want 0", tt.ts, result)
			}
			if !tt.wantZero && result == 0 {
				t.Errorf("parseTimestamp(%q) = 0, want non-zero", tt.ts)
			}
		})
	}
}
