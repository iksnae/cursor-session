package internal

import (
	"fmt"
	"time"
)

// Normalizer converts reconstructed conversations to Session format
type Normalizer struct{}

// NewNormalizer creates a new Normalizer
func NewNormalizer() *Normalizer {
	return &Normalizer{}
}

// NormalizeConversation converts a ReconstructedConversation to a Session
func (n *Normalizer) NormalizeConversation(conv *ReconstructedConversation, workspace string) (*Session, error) {
	if conv == nil {
		return nil, fmt.Errorf("conversation is nil")
	}

	if len(conv.Messages) == 0 {
		return nil, fmt.Errorf("conversation has no messages")
	}

	// Use composerId as the session ID (the real session identifier from Cursor)
	sessionID := conv.ComposerID

	// Convert messages
	messages := make([]Message, 0, len(conv.Messages))
	for _, msg := range conv.Messages {
		normalizedMsg := n.normalizeMessage(msg)
		messages = append(messages, normalizedMsg)
	}

	// Create metadata
	metadata := Metadata{
		ComposerID:   conv.ComposerID,
		Name:         conv.Name,
		MessageCount: len(messages),
	}

	if conv.CreatedAt > 0 {
		metadata.CreatedAt = formatTimestamp(conv.CreatedAt)
	}
	if conv.UpdatedAt > 0 {
		metadata.UpdatedAt = formatTimestamp(conv.UpdatedAt)
	}

	return &Session{
		ID:        sessionID,
		Workspace: workspace,
		Source:    "globalStorage",
		Messages:  messages,
		Metadata:  metadata,
	}, nil
}

// normalizeMessage converts a ReconstructedMessage to a Message
func (n *Normalizer) normalizeMessage(msg ReconstructedMessage) Message {
	actor := n.normalizeActor(msg.Type)
	timestamp := ""
	if msg.Timestamp > 0 {
		timestamp = formatTimestamp(msg.Timestamp)
	}

	return Message{
		Timestamp: timestamp,
		Actor:     actor,
		Content:   msg.Text,
	}
}

// normalizeActor converts type (1 or 2) to actor string
func (n *Normalizer) normalizeActor(msgType int) string {
	switch msgType {
	case 1:
		return "user"
	case 2:
		return "assistant"
	default:
		return "user" // Default fallback
	}
}

// generateSessionID is deprecated - we now use composerId directly as the session ID
// This function is kept for backwards compatibility but should not be used
func (n *Normalizer) generateSessionID(conv *ReconstructedConversation) string {
	// Just return the composerId - this maintains compatibility
	return conv.ComposerID
}

// formatTimestamp formats a Unix timestamp (milliseconds) to ISO8601
func formatTimestamp(ts int64) string {
	t := time.Unix(0, ts*int64(time.Millisecond))
	return t.Format(time.RFC3339)
}

// NormalizeAllConversations normalizes all conversations to sessions
func (n *Normalizer) NormalizeAllConversations(conversations []*ReconstructedConversation, workspace string) ([]*Session, error) {
	var sessions []*Session

	for _, conv := range conversations {
		session, err := n.NormalizeConversation(conv, workspace)
		if err != nil {
			// Log error but continue
			continue
		}
		sessions = append(sessions, session)
	}

	return sessions, nil
}
