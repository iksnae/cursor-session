package internal

import (
	"os"
	"strings"
	"testing"

	"github.com/iksnae/cursor-session/testutil"
)

func TestNewStorage(t *testing.T) {
	db := testutil.CreateInMemoryDB(t)
	defer db.Close()

	storage := NewStorage(db)
	// NewStorage always returns a non-nil pointer
	//nolint:staticcheck // SA5011: false positive - NewStorage never returns nil
	if storage.db != db {
		t.Error("NewStorage() did not set database correctly")
	}
}

func TestStorage_LoadBubbles(t *testing.T) {
	db := testutil.CreateTestDB(t)
	defer db.Close()

	storage := NewStorage(db)
	bubbles, err := storage.LoadBubbles()
	if err != nil {
		t.Fatalf("LoadBubbles() error = %v", err)
	}

	if len(bubbles) == 0 {
		t.Error("LoadBubbles() returned empty map")
	}

	// Verify bubble structure
	for bubbleID, bubble := range bubbles {
		if bubbleID != bubble.BubbleID {
			t.Errorf("Bubble map key %q does not match BubbleID %q", bubbleID, bubble.BubbleID)
		}
		if bubble.ChatID == "" {
			t.Error("Bubble ChatID should not be empty")
		}
	}
}

func TestStorage_LoadBubbles_InvalidData(t *testing.T) {
	db := testutil.CreateInMemoryDB(t)
	defer db.Close()

	// Insert invalid bubble data
	testutil.InsertBubble(t, db, "bubbleId:chat1:invalid", "not valid json")

	storage := NewStorage(db)
	bubbles, err := storage.LoadBubbles()
	if err != nil {
		t.Fatalf("LoadBubbles() error = %v", err)
	}

	// Should skip invalid data and continue
	if bubbles == nil {
		t.Error("LoadBubbles() should return map even with invalid data")
	}
}

func TestStorage_LoadComposers(t *testing.T) {
	db := testutil.CreateTestDB(t)
	defer db.Close()

	storage := NewStorage(db)
	composers, err := storage.LoadComposers()
	if err != nil {
		t.Fatalf("LoadComposers() error = %v", err)
	}

	if len(composers) == 0 {
		t.Error("LoadComposers() returned empty slice")
	}

	// Verify composer structure
	for _, composer := range composers {
		if composer.ComposerID == "" {
			t.Error("Composer ComposerID should not be empty")
		}
	}
}

func TestStorage_LoadComposers_InvalidData(t *testing.T) {
	db := testutil.CreateInMemoryDB(t)
	defer db.Close()

	// Insert invalid composer data
	testutil.InsertComposer(t, db, "composerData:invalid", "not valid json")

	storage := NewStorage(db)
	composers, err := storage.LoadComposers()
	if err != nil {
		t.Fatalf("LoadComposers() error = %v", err)
	}

	// Should skip invalid data and continue
	if composers == nil {
		t.Error("LoadComposers() should return slice even with invalid data")
	}
}

func TestStorage_LoadMessageContexts(t *testing.T) {
	db := testutil.CreateTestDB(t)
	defer db.Close()

	storage := NewStorage(db)
	contexts, err := storage.LoadMessageContexts()
	if err != nil {
		t.Fatalf("LoadMessageContexts() error = %v", err)
	}

	if len(contexts) == 0 {
		t.Error("LoadMessageContexts() returned empty map")
	}

	// Verify context structure
	for composerID, ctxList := range contexts {
		if composerID == "" {
			t.Error("Context map key (ComposerID) should not be empty")
		}
		for _, ctx := range ctxList {
			if ctx.ComposerID != composerID {
				t.Errorf("Context ComposerID %q does not match map key %q", ctx.ComposerID, composerID)
			}
		}
	}
}

func TestStorage_LoadMessageContexts_InvalidData(t *testing.T) {
	db := testutil.CreateInMemoryDB(t)
	defer db.Close()

	// Insert invalid context data
	testutil.InsertBubble(t, db, "messageRequestContext:composer1:invalid", "not valid json")

	storage := NewStorage(db)
	contexts, err := storage.LoadMessageContexts()
	if err != nil {
		t.Fatalf("LoadMessageContexts() error = %v", err)
	}

	// Should skip invalid data and continue
	if contexts == nil {
		t.Error("LoadMessageContexts() should return map even with invalid data")
	}
}

func TestStorage_LoadCodeBlockDiffs(t *testing.T) {
	db := testutil.CreateInMemoryDB(t)
	defer db.Close()

	// Insert code block diff data
	diffData := `{"type":"diff","content":"test"}`
	testutil.InsertBubble(t, db, "codeBlockDiff:chat1:diff1", diffData)

	storage := NewStorage(db)
	diffs, err := storage.LoadCodeBlockDiffs()
	if err != nil {
		t.Fatalf("LoadCodeBlockDiffs() error = %v", err)
	}

	if len(diffs) == 0 {
		t.Error("LoadCodeBlockDiffs() returned empty map")
	}

	// Verify diff structure
	for chatID, diffList := range diffs {
		if chatID == "" {
			t.Error("Diff map key (ChatID) should not be empty")
		}
		if len(diffList) == 0 {
			t.Error("Diff list should not be empty")
		}
	}
}

func TestStorage_LoadCodeBlockDiffs_InvalidKey(t *testing.T) {
	db := testutil.CreateInMemoryDB(t)
	defer db.Close()

	// Insert code block diff with invalid key format
	testutil.InsertBubble(t, db, "codeBlockDiff:invalid", `{"type":"diff"}`)

	storage := NewStorage(db)
	diffs, err := storage.LoadCodeBlockDiffs()
	if err != nil {
		t.Fatalf("LoadCodeBlockDiffs() error = %v", err)
	}

	// Should skip invalid keys
	if diffs == nil {
		t.Error("LoadCodeBlockDiffs() should return map even with invalid keys")
	}
}

func TestStorage_LoadCodeBlockDiffs_InvalidJSON(t *testing.T) {
	db := testutil.CreateInMemoryDB(t)
	defer db.Close()

	// Insert code block diff with invalid JSON
	testutil.InsertBubble(t, db, "codeBlockDiff:chat1:diff1", "not valid json")

	storage := NewStorage(db)
	diffs, err := storage.LoadCodeBlockDiffs()
	if err != nil {
		t.Fatalf("LoadCodeBlockDiffs() error = %v", err)
	}

	// Should skip invalid JSON
	if diffs == nil {
		t.Error("LoadCodeBlockDiffs() should return map even with invalid JSON")
	}
}

func TestStorage_ImplementsStorageBackend(t *testing.T) {
	// Test that Storage implements StorageBackend interface
	var _ StorageBackend = (*Storage)(nil)
}

func TestAgentStorage_ImplementsStorageBackend(t *testing.T) {
	// Test that AgentStorage implements StorageBackend interface
	var _ StorageBackend = (*AgentStorage)(nil)
}

func TestNewAgentStorage(t *testing.T) {
	paths := []string{"/path1/store.db", "/path2/store.db"}
	agentStorage := NewAgentStorage(paths)

	if agentStorage == nil {
		t.Fatal("NewAgentStorage() returned nil")
	}

	if agentStorage.reader == nil {
		t.Error("NewAgentStorage() did not initialize reader")
	}
}

func TestAgentStorage_LoadCodeBlockDiffs(t *testing.T) {
	agentStorage := NewAgentStorage([]string{})
	diffs, err := agentStorage.LoadCodeBlockDiffs()
	if err != nil {
		t.Fatalf("LoadCodeBlockDiffs() error = %v", err)
	}

	if diffs == nil {
		t.Error("LoadCodeBlockDiffs() should return a map, not nil")
	}

	if len(diffs) != 0 {
		t.Errorf("LoadCodeBlockDiffs() returned %d diffs, want 0", len(diffs))
	}
}

func TestNewStorageBackend_DesktopAppFormat(t *testing.T) {
	// This test requires a real database file, so we'll skip it if not available
	// In a real scenario, we'd use test fixtures
	paths, err := DetectStoragePaths()
	if err != nil {
		t.Fatalf("DetectStoragePaths() error = %v", err)
	}

	// Only test if globalStorage exists
	if paths.GlobalStorageExists() {
		backend, err := NewStorageBackend(paths)
		if err != nil {
			t.Fatalf("NewStorageBackend() error = %v", err)
		}

		if backend == nil {
			t.Fatal("NewStorageBackend() returned nil")
		}

		// Verify it's a Storage instance (desktop app format)
		if _, ok := backend.(*Storage); !ok {
			t.Error("NewStorageBackend() should return *Storage when globalStorage exists")
		}
	}
}

func TestNewStorageBackend_NoStorageAvailable(t *testing.T) {
	// Create paths with nonexistent storage
	testPaths := StoragePaths{
		GlobalStorage:    "/nonexistent/globalStorage",
		AgentStoragePath: "/nonexistent/.cursor/chats",
	}

	backend, err := NewStorageBackend(testPaths)
	if err == nil {
		t.Error("NewStorageBackend() should return error when no storage is available")
	}

	if backend != nil {
		t.Error("NewStorageBackend() should return nil backend on error")
	}

	// Verify error message contains helpful information
	errMsg := err.Error()
	if !strings.Contains(errMsg, "no Cursor storage found") {
		t.Errorf("Error message should mention 'no Cursor storage found', got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "Checked storage locations") {
		t.Errorf("Error message should mention checked locations, got: %s", errMsg)
	}
}

func TestIsCIEnvironment(t *testing.T) {
	// Save original environment
	originalCI := os.Getenv("CI")
	originalGHA := os.Getenv("GITHUB_ACTIONS")
	
	// Clean up after test
	defer func() {
		if originalCI != "" {
			os.Setenv("CI", originalCI)
		} else {
			os.Unsetenv("CI")
		}
		if originalGHA != "" {
			os.Setenv("GITHUB_ACTIONS", originalGHA)
		} else {
			os.Unsetenv("GITHUB_ACTIONS")
		}
	}()

	// Test with CI environment variable set
	os.Setenv("CI", "true")
	if !IsCIEnvironment() {
		t.Error("IsCIEnvironment() should return true when CI is set")
	}

	// Test with GitHub Actions environment variable set
	os.Unsetenv("CI")
	os.Setenv("GITHUB_ACTIONS", "true")
	if !IsCIEnvironment() {
		t.Error("IsCIEnvironment() should return true when GITHUB_ACTIONS is set")
	}

	// Test without CI environment variables
	os.Unsetenv("CI")
	os.Unsetenv("GITHUB_ACTIONS")
	// Note: We can't easily test false case without clearing all CI vars,
	// but the function should work correctly
}


