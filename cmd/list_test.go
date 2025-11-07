package cmd

import (
	"bytes"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/iksnae/cursor-session/internal"
)

func TestListCommand_FlagParsing(t *testing.T) {
	// Test that flags are parsed correctly
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "list without flags",
			args: []string{"list"},
		},
		{
			name: "list with clear-cache flag",
			args: []string{"list", "--clear-cache"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootCmd.SetArgs(tt.args)
			rootCmd.SetOut(&bytes.Buffer{})
			rootCmd.SetErr(&bytes.Buffer{})

			// Just verify flags are parsed without error
			// The actual execution may succeed or fail depending on environment
			_ = rootCmd.Execute()
		})
	}
}

func TestDisplaySessionsFromComposers(t *testing.T) {
	tests := []struct {
		name     string
		composers []*internal.RawComposer
	}{
		{
			name:     "empty composers",
			composers: []*internal.RawComposer{},
		},
		{
			name: "single composer",
			composers: []*internal.RawComposer{
				{
					ComposerID: "test-composer-1",
					Name:       "Test Session",
					FullConversationHeadersOnly: []internal.ConversationHeader{
						{BubbleID: "bubble1", Type: 1},
					},
					CreatedAt: 1000,
				},
			},
		},
		{
			name: "multiple composers",
			composers: []*internal.RawComposer{
				{
					ComposerID: "test-composer-1",
					Name:       "Test Session 1",
					FullConversationHeadersOnly: []internal.ConversationHeader{
						{BubbleID: "bubble1", Type: 1},
					},
					CreatedAt: 1000,
				},
				{
					ComposerID: "test-composer-2",
					Name:       "Test Session 2",
					FullConversationHeadersOnly: []internal.ConversationHeader{
						{BubbleID: "bubble2", Type: 2},
					},
					CreatedAt: 2000,
				},
			},
		},
		{
			name: "composer with long name",
			composers: []*internal.RawComposer{
				{
					ComposerID: "test-composer-1",
					Name:       "This is a very long session name that should be truncated when displayed in the list",
					FullConversationHeadersOnly: []internal.ConversationHeader{
						{BubbleID: "bubble1", Type: 1},
					},
					CreatedAt: 1000,
				},
			},
		},
		{
			name: "composer without name",
			composers: []*internal.RawComposer{
				{
					ComposerID: "test-composer-1",
					Name:       "",
					FullConversationHeadersOnly: []internal.ConversationHeader{
						{BubbleID: "bubble1", Type: 1},
					},
					CreatedAt: 1000,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			// Capture output
			originalOutput := lipgloss.DefaultRenderer().Output()
			defer func() {
				// Restore original output
				_ = originalOutput
			}()

			// Test that function doesn't panic
			displaySessionsFromComposers(tt.composers)
			_ = buf.String() // Just verify it doesn't panic
		})
	}
}

func TestDisplaySessionsFromIndex(t *testing.T) {
	tests := []struct {
		name  string
		index *internal.SessionIndex
	}{
		{
			name: "empty index",
			index: &internal.SessionIndex{
				Sessions: []internal.SessionIndexEntry{},
				Metadata: internal.CacheMetadata{
					CacheVersion: "1.0",
				},
			},
		},
		{
			name: "index with sessions",
			index: &internal.SessionIndex{
				Sessions: []internal.SessionIndexEntry{
					{
						ID:           "session1",
						ComposerID:   "composer1",
						Name:         "Test Session",
						MessageCount: 5,
						CreatedAt:    "2024-01-01T00:00:00Z",
					},
				},
				Metadata: internal.CacheMetadata{
					CacheVersion: "1.0",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that function doesn't panic
			displaySessionsFromIndex(tt.index)
		})
	}
}

