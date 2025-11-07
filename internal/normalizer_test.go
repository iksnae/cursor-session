package internal

import (
	"testing"
)

func TestNormalizeActor(t *testing.T) {
	normalizer := NewNormalizer()

	tests := []struct {
		input int
		want  string
	}{
		{1, "user"},
		{2, "assistant"},
		{0, "user"}, // default
		{3, "user"}, // default
	}

	for _, tt := range tests {
		got := normalizer.normalizeActor(tt.input)
		if got != tt.want {
			t.Errorf("normalizeActor(%d) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestGenerateSessionID(t *testing.T) {
	normalizer := NewNormalizer()

	conv := &ReconstructedConversation{
		ComposerID: "test-composer-id",
		Messages: []ReconstructedMessage{
			{Text: "Hello", Timestamp: 1000},
		},
	}

	sessionID := normalizer.generateSessionID(conv)
	if sessionID == "" {
		t.Error("generateSessionID() should not return empty string")
	}

	// Should be stable for same input
	sessionID2 := normalizer.generateSessionID(conv)
	if sessionID != sessionID2 {
		t.Error("generateSessionID() should be stable for same input")
	}
}

func TestNormalizeConversation(t *testing.T) {
	normalizer := NewNormalizer()

	tests := []struct {
		name    string
		conv    *ReconstructedConversation
		wantErr bool
	}{
		{
			name:    "nil conversation",
			conv:    nil,
			wantErr: true,
		},
		{
			name: "empty messages",
			conv: &ReconstructedConversation{
				ComposerID: "composer1",
				Messages:   []ReconstructedMessage{},
			},
			wantErr: true,
		},
		{
			name: "valid conversation",
			conv: &ReconstructedConversation{
				ComposerID: "composer1",
				Name:       "Test",
				Messages: []ReconstructedMessage{
					{Type: 1, Text: "Hello", Timestamp: 1000},
					{Type: 2, Text: "Hi", Timestamp: 2000},
				},
				CreatedAt: 1000,
				UpdatedAt: 2000,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session, err := normalizer.NormalizeConversation(tt.conv, "workspace1")
			if (err != nil) != tt.wantErr {
				t.Errorf("NormalizeConversation() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if session == nil {
					t.Error("NormalizeConversation() returned nil session")
					return
				}

				if session.ID != tt.conv.ComposerID {
					t.Errorf("Session.ID = %q, want %q", session.ID, tt.conv.ComposerID)
				}

				if len(session.Messages) != len(tt.conv.Messages) {
					t.Errorf("Session.Messages length = %d, want %d", len(session.Messages), len(tt.conv.Messages))
				}
			}
		})
	}
}

func TestNormalizeAllConversations(t *testing.T) {
	normalizer := NewNormalizer()

	conversations := []*ReconstructedConversation{
		{
			ComposerID: "composer1",
			Messages: []ReconstructedMessage{
				{Type: 1, Text: "Hello", Timestamp: 1000},
			},
		},
		{
			ComposerID: "composer2",
			Messages: []ReconstructedMessage{
				{Type: 2, Text: "Hi", Timestamp: 2000},
			},
		},
		{
			ComposerID: "composer3",
			Messages:   []ReconstructedMessage{}, // Empty, should be skipped
		},
	}

	sessions, err := normalizer.NormalizeAllConversations(conversations, "workspace1")
	if err != nil {
		t.Fatalf("NormalizeAllConversations() error = %v", err)
	}

	// Should only return sessions with messages
	if len(sessions) != 2 {
		t.Errorf("NormalizeAllConversations() returned %d sessions, want 2", len(sessions))
	}
}

func TestFormatTimestamp(t *testing.T) {
	// Test formatTimestamp indirectly through NormalizeConversation
	normalizer := NewNormalizer()

	conv := &ReconstructedConversation{
		ComposerID: "composer1",
		Messages: []ReconstructedMessage{
			{Type: 1, Text: "Hello", Timestamp: 1000},
		},
		CreatedAt: 1000,
		UpdatedAt: 2000,
	}

	session, err := normalizer.NormalizeConversation(conv, "workspace1")
	if err != nil {
		t.Fatalf("NormalizeConversation() error = %v", err)
	}

	if session.Metadata.CreatedAt == "" {
		t.Error("Session.Metadata.CreatedAt should not be empty")
	}

	if session.Metadata.UpdatedAt == "" {
		t.Error("Session.Metadata.UpdatedAt should not be empty")
	}
}
