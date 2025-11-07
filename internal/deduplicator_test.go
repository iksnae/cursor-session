package internal

import (
	"testing"

)

func TestNewDeduplicator(t *testing.T) {
	d := NewDeduplicator()
	if d == nil {
		t.Error("NewDeduplicator() returned nil")
	}
}

func TestDeduplicator_Deduplicate(t *testing.T) {
	tests := []struct {
		name     string
		sessions []*Session
		want     int
	}{
		{
			name:     "empty sessions",
			sessions: []*Session{},
			want:     0,
		},
		{
			name: "no duplicates",
			sessions: []*Session{
				CreateTestSessionWithMessages("session1", []Message{
					{Actor: "user", Content: "Hello"},
				}),
				CreateTestSessionWithMessages("session2", []Message{
					{Actor: "user", Content: "Goodbye"},
				}),
			},
			want: 2,
		},
		{
			name: "with duplicates",
			sessions: []*Session{
				CreateTestSessionWithMessages("session1", []Message{
					{Actor: "user", Content: "Hello"},
				}),
				CreateTestSessionWithMessages("session1-dup", []Message{
					{Actor: "user", Content: "Hello"}, // same content = duplicate
				}),
				CreateTestSessionWithMessages("session2", []Message{
					{Actor: "user", Content: "Goodbye"},
				}),
			},
			want: 2,
		},
		{
			name: "all duplicates",
			sessions: []*Session{
				CreateTestSessionWithMessages("session1", []Message{
					{Actor: "user", Content: "Hello"},
				}),
				CreateTestSessionWithMessages("session1-dup1", []Message{
					{Actor: "user", Content: "Hello"}, // same content
				}),
				CreateTestSessionWithMessages("session1-dup2", []Message{
					{Actor: "user", Content: "Hello"}, // same content
				}),
			},
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDeduplicator()
			got := d.Deduplicate(tt.sessions)

			if len(got) != tt.want {
				t.Errorf("Deduplicate() returned %d sessions, want %d", len(got), tt.want)
			}
		})
	}
}

func TestDeduplicator_Deduplicate_ContentBased(t *testing.T) {
	// Create sessions with same content but different IDs
	session1 := CreateTestSessionWithMessages("session1", []Message{
		{Actor: "user", Content: "Hello"},
		{Actor: "assistant", Content: "Hi"},
	})

	session2 := CreateTestSessionWithMessages("session2", []Message{
		{Actor: "user", Content: "Hello"},
		{Actor: "assistant", Content: "Hi"},
	})

	// These should be considered duplicates (same content)
	sessions := []*Session{session1, session2}

	d := NewDeduplicator()
	got := d.Deduplicate(sessions)

	if len(got) != 1 {
		t.Errorf("Deduplicate() returned %d sessions, want 1 (content-based deduplication)", len(got))
	}
}

func TestDeduplicator_Deduplicate_DifferentContent(t *testing.T) {
	// Create sessions with different content
	session1 := CreateTestSessionWithMessages("session1", []Message{
		{Actor: "user", Content: "Hello"},
	})

	session2 := CreateTestSessionWithMessages("session2", []Message{
		{Actor: "user", Content: "Goodbye"},
	})

	sessions := []*Session{session1, session2}

	d := NewDeduplicator()
	got := d.Deduplicate(sessions)

	if len(got) != 2 {
		t.Errorf("Deduplicate() returned %d sessions, want 2 (different content)", len(got))
	}
}

func TestDeduplicator_hashSessionContent(t *testing.T) {
	d := NewDeduplicator()

	session1 := CreateTestSessionWithMessages("session1", []Message{
		{Actor: "user", Content: "Hello"},
	})
	hash1 := d.hashSessionContent(session1)

	// Same session should produce same hash
	hash2 := d.hashSessionContent(session1)
	if hash1 != hash2 {
		t.Error("hashSessionContent() should be stable for same session")
	}

	// Different session with different content should produce different hash
	session2 := CreateTestSessionWithMessages("session2", []Message{
		{Actor: "user", Content: "Goodbye"},
	})
	hash3 := d.hashSessionContent(session2)
	if hash1 == hash3 {
		t.Error("hashSessionContent() should produce different hashes for different sessions")
	}
}


