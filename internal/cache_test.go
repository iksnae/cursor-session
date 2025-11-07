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
		t.Error("NewCacheManager() returned nil")
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
	cm.EnsureCacheDir()

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
				cm.SaveIndex(index)
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
				cm.SaveIndex(index)
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
				cm.SaveIndex(index)
			},
			want:    false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up previous cache
			os.Remove(cm.GetIndexPath())
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
	cm.EnsureCacheDir()

	index := &SessionIndex{
		Sessions: []SessionIndexEntry{
			{
				ID:           "session1",
				ComposerID:   "composer1",
				Name:        "Test Session",
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
	cm.EnsureCacheDir()

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
	cm.EnsureCacheDir()

	session1 := CreateTestSession("session1")
	session2 := CreateTestSession("session2")

	cm.SaveSession(session1)
	cm.SaveSession(session2)

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
	cm.SaveIndex(index)

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


