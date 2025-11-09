package internal

import (
	"testing"
)

func TestNewReconstructor(t *testing.T) {
	bubbleMap := NewBubbleMap()
	contextMap := make(map[string][]*MessageContext)

	reconstructor := NewReconstructor(bubbleMap, contextMap)
	// NewReconstructor always returns a non-nil pointer
	//nolint:staticcheck // SA5011: false positive - NewReconstructor never returns nil
	if reconstructor.bubbleMap != bubbleMap {
		t.Error("NewReconstructor() did not set bubbleMap correctly")
	}
	// Can't compare maps directly, so check they're both non-nil or both nil
	//nolint:staticcheck // SA5011: false positive
	if (reconstructor.contextMap == nil) != (contextMap == nil) {
		t.Error("NewReconstructor() did not set contextMap correctly")
	}
	//nolint:staticcheck // SA5011: false positive
	if len(reconstructor.contextMap) != len(contextMap) {
		t.Error("NewReconstructor() did not set contextMap correctly")
	}
}

func TestReconstructor_ReconstructConversation(t *testing.T) {
	bubbleMap := NewBubbleMap()
	bubble1 := CreateTestRawBubble("bubble1", "chat1", "Hello", 1)
	bubble2 := CreateTestRawBubble("bubble2", "chat1", "Hi there", 2)
	bubbleMap.Set("bubble1", bubble1)
	bubbleMap.Set("bubble2", bubble2)

	composer := &RawComposer{
		ComposerID: "composer1",
		Name:       "Test Conversation",
		FullConversationHeadersOnly: []ConversationHeader{
			{BubbleID: "bubble1", Type: 1},
			{BubbleID: "bubble2", Type: 2},
		},
		CreatedAt:     1000,
		LastUpdatedAt: 2000,
	}

	contextMap := make(map[string][]*MessageContext)
	reconstructor := NewReconstructor(bubbleMap, contextMap)

	conv, err := reconstructor.ReconstructConversation(composer)
	if err != nil {
		t.Fatalf("ReconstructConversation() error = %v", err)
	}

	if conv == nil {
		t.Fatal("ReconstructConversation() returned nil")
	}

	if conv.ComposerID != "composer1" {
		t.Errorf("ReconstructConversation() ComposerID = %q, want composer1", conv.ComposerID)
	}

	if len(conv.Messages) != 2 {
		t.Errorf("ReconstructConversation() returned %d messages, want 2", len(conv.Messages))
	}

	// Verify messages are sorted by timestamp
	if conv.Messages[0].Timestamp > conv.Messages[1].Timestamp {
		t.Error("ReconstructConversation() messages should be sorted by timestamp")
	}
}

func TestReconstructor_ReconstructConversation_NilComposer(t *testing.T) {
	bubbleMap := NewBubbleMap()
	contextMap := make(map[string][]*MessageContext)
	reconstructor := NewReconstructor(bubbleMap, contextMap)

	_, err := reconstructor.ReconstructConversation(nil)
	if err == nil {
		t.Error("ReconstructConversation() should return error for nil composer")
	}
}

func TestReconstructor_ReconstructConversation_MissingBubble(t *testing.T) {
	bubbleMap := NewBubbleMap()
	composer := &RawComposer{
		ComposerID: "composer1",
		FullConversationHeadersOnly: []ConversationHeader{
			{BubbleID: "nonexistent", Type: 1},
		},
	}

	contextMap := make(map[string][]*MessageContext)
	reconstructor := NewReconstructor(bubbleMap, contextMap)

	conv, err := reconstructor.ReconstructConversation(composer)
	if err != nil {
		t.Fatalf("ReconstructConversation() error = %v", err)
	}

	// Should skip missing bubble and return empty messages
	if len(conv.Messages) != 0 {
		t.Errorf("ReconstructConversation() returned %d messages, want 0 (missing bubble)", len(conv.Messages))
	}
}

func TestReconstructor_ReconstructAllConversations(t *testing.T) {
	bubbleMap := NewBubbleMap()
	bubble1 := CreateTestRawBubble("bubble1", "chat1", "Hello", 1)
	bubbleMap.Set("bubble1", bubble1)

	composers := []*RawComposer{
		{
			ComposerID: "composer1",
			FullConversationHeadersOnly: []ConversationHeader{
				{BubbleID: "bubble1", Type: 1},
			},
		},
		{
			ComposerID: "composer2",
			FullConversationHeadersOnly: []ConversationHeader{
				{BubbleID: "nonexistent", Type: 1}, // Will be skipped
			},
		},
	}

	contextMap := make(map[string][]*MessageContext)
	reconstructor := NewReconstructor(bubbleMap, contextMap)

	conversations, err := reconstructor.ReconstructAllConversations(composers)
	if err != nil {
		t.Fatalf("ReconstructAllConversations() error = %v", err)
	}

	// Should only return conversations with messages
	if len(conversations) != 1 {
		t.Errorf("ReconstructAllConversations() returned %d conversations, want 1", len(conversations))
	}

	if conversations[0].ComposerID != "composer1" {
		t.Errorf("ReconstructAllConversations() ComposerID = %q, want composer1", conversations[0].ComposerID)
	}
}

func TestReconstructor_ReconstructAllConversations_Empty(t *testing.T) {
	bubbleMap := NewBubbleMap()
	contextMap := make(map[string][]*MessageContext)
	reconstructor := NewReconstructor(bubbleMap, contextMap)

	conversations, err := reconstructor.ReconstructAllConversations([]*RawComposer{})
	if err != nil {
		t.Fatalf("ReconstructAllConversations() error = %v", err)
	}

	if len(conversations) != 0 {
		t.Errorf("ReconstructAllConversations() returned %d conversations, want 0", len(conversations))
	}
}
