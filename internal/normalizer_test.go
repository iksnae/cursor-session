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

