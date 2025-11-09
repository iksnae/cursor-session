package internal

import (
	"time"
)

// CreateTestSession creates a test session with sample data
func CreateTestSession(id string) *Session {
	return &Session{
		ID:        id,
		Workspace: "test-workspace",
		Source:    "globalStorage",
		Messages: []Message{
			{
				Actor:     "user",
				Content:   "Hello, how are you?",
				Timestamp: time.Now().Format(time.RFC3339),
			},
			{
				Actor:     "assistant",
				Content:   "I'm doing well, thank you!",
				Timestamp: time.Now().Format(time.RFC3339),
			},
		},
		Metadata: Metadata{
			Name:         "Test Conversation",
			ComposerID:   "composer-" + id,
			MessageCount: 2,
			CreatedAt:    time.Now().Format(time.RFC3339),
		},
	}
}

// CreateTestSessionWithMessages creates a test session with custom messages
func CreateTestSessionWithMessages(id string, messages []Message) *Session {
	return &Session{
		ID:        id,
		Workspace: "test-workspace",
		Source:    "globalStorage",
		Messages:  messages,
		Metadata: Metadata{
			ComposerID:   "composer-" + id,
			MessageCount: len(messages),
		},
	}
}

// CreateTestRawBubble creates a test RawBubble
func CreateTestRawBubble(bubbleID, chatID, text string, msgType int) *RawBubble {
	return &RawBubble{
		BubbleID:  bubbleID,
		ChatID:    chatID,
		Text:      text,
		Timestamp: time.Now().UnixMilli(),
		Type:      msgType,
	}
}

// CreateTestRawComposer creates a test RawComposer
func CreateTestRawComposer(composerID, name string) *RawComposer {
	now := time.Now().UnixMilli()
	return &RawComposer{
		ComposerID:    composerID,
		Name:          name,
		CreatedAt:     now,
		LastUpdatedAt: now,
	}
}

// CreateTestMessageContext creates a test MessageContext
func CreateTestMessageContext(bubbleID, composerID, contextID string) *MessageContext {
	return &MessageContext{
		BubbleID:   bubbleID,
		ComposerID: composerID,
		ContextID:  contextID,
	}
}
