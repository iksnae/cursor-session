package internal

import (
	"testing"

)

func TestNewBubbleMap(t *testing.T) {
	bm := NewBubbleMap()
	if bm == nil {
		t.Error("NewBubbleMap() returned nil")
	}
	if bm.bubbles == nil {
		t.Error("NewBubbleMap() bubbles map is nil")
	}
}

func TestBubbleMap_GetSet(t *testing.T) {
	bm := NewBubbleMap()

	bubble := CreateTestRawBubble("bubble1", "chat1", "Hello", 1)

	// Test Set
	bm.Set("bubble1", bubble)

	// Test Get
	got, ok := bm.Get("bubble1")
	if !ok {
		t.Error("Get() returned false for existing bubble")
	}
	if got != bubble {
		t.Errorf("Get() = %v, want %v", got, bubble)
	}

	// Test Get non-existent
	_, ok = bm.Get("nonexistent")
	if ok {
		t.Error("Get() returned true for non-existent bubble")
	}
}

func TestBubbleMap_ConcurrentAccess(t *testing.T) {
	bm := NewBubbleMap()

	// Test concurrent writes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			bubble := CreateTestRawBubble("bubble"+string(rune(id)), "chat1", "Hello", 1)
			bm.Set("bubble"+string(rune(id)), bubble)
			done <- true
		}(i)
	}

	// Wait for all writes
	for i := 0; i < 10; i++ {
		<-done
	}

	// Test concurrent reads
	for i := 0; i < 10; i++ {
		go func(id int) {
			_, _ = bm.Get("bubble" + string(rune(id)))
			done <- true
		}(i)
	}

	// Wait for all reads
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestBuildBubbleMapFromChannel(t *testing.T) {
	bubbleChan := make(chan *RawBubble, 3)

	bubble1 := CreateTestRawBubble("bubble1", "chat1", "Hello", 1)
	bubble2 := CreateTestRawBubble("bubble2", "chat1", "Hi", 2)
	bubble3 := CreateTestRawBubble("bubble3", "chat2", "Hey", 1)

	bubbleChan <- bubble1
	bubbleChan <- bubble2
	bubbleChan <- bubble3
	close(bubbleChan)

	bm := BuildBubbleMapFromChannel(bubbleChan)

	if bm == nil {
		t.Error("BuildBubbleMapFromChannel() returned nil")
	}

	// Verify all bubbles are in the map
	if got, ok := bm.Get("bubble1"); !ok || got != bubble1 {
		t.Error("Bubble1 not found in map")
	}
	if got, ok := bm.Get("bubble2"); !ok || got != bubble2 {
		t.Error("Bubble2 not found in map")
	}
	if got, ok := bm.Get("bubble3"); !ok || got != bubble3 {
		t.Error("Bubble3 not found in map")
	}
}

func TestBuildBubbleMapFromChannel_NilBubbles(t *testing.T) {
	bubbleChan := make(chan *RawBubble, 2)

	bubble1 := CreateTestRawBubble("bubble1", "chat1", "Hello", 1)
	bubbleChan <- bubble1
	bubbleChan <- nil // nil bubble should be skipped
	close(bubbleChan)

	bm := BuildBubbleMapFromChannel(bubbleChan)

	if got, ok := bm.Get("bubble1"); !ok || got != bubble1 {
		t.Error("Bubble1 not found in map")
	}

	// Verify nil bubble was not added
	if len(bm.bubbles) != 1 {
		t.Errorf("Map should have 1 bubble, got %d", len(bm.bubbles))
	}
}

func TestBuildBubbleMapFromChannel_EmptyChannel(t *testing.T) {
	bubbleChan := make(chan *RawBubble)
	close(bubbleChan)

	bm := BuildBubbleMapFromChannel(bubbleChan)

	if bm == nil {
		t.Error("BuildBubbleMapFromChannel() returned nil for empty channel")
	}

	if len(bm.bubbles) != 0 {
		t.Errorf("Map should be empty, got %d bubbles", len(bm.bubbles))
	}
}

