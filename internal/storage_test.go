package internal

import (
	"testing"

	"github.com/iksnae/cursor-session/testutil"
)

func TestNewStorage(t *testing.T) {
	db := testutil.CreateInMemoryDB(t)
	defer db.Close()

	storage := NewStorage(db)
	if storage == nil {
		t.Error("NewStorage() returned nil")
	}
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


