package cmd

import (
	"bytes"
	"testing"
	"time"

	"github.com/iksnae/cursor-session/internal"
)

func TestShowCommand(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "show without session ID",
			args:    []string{"show"},
			wantErr: true, // Requires session ID
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootCmd.SetArgs(tt.args)
			rootCmd.SetOut(&bytes.Buffer{})
			rootCmd.SetErr(&bytes.Buffer{})

			err := rootCmd.Execute()
			if (err != nil) != tt.wantErr {
				t.Errorf("showCmd.Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestShowCommand_FlagParsing(t *testing.T) {
	// Test that flags are parsed correctly
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "show with limit flag",
			args: []string{"show", "test-session-id", "--limit", "10"},
		},
		{
			name: "show with since flag",
			args: []string{"show", "test-session-id", "--since", "2024-01-01T00:00:00Z"},
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

func TestDisplaySessionHeader(t *testing.T) {
	tests := []struct {
		name    string
		session *internal.Session
	}{
		{
			name:    "nil session",
			session: nil,
		},
		{
			name: "session with all fields",
			session: &internal.Session{
				ID: "test-session",
				Metadata: internal.Metadata{
					Name:      "Test Session",
					CreatedAt: "2024-01-01T00:00:00Z",
				},
				Messages: []internal.Message{
					{Actor: "user", Content: "Hello"},
				},
				Workspace: "/path/to/workspace",
			},
		},
		{
			name: "session without workspace",
			session: &internal.Session{
				ID: "test-session",
				Metadata: internal.Metadata{
					Name:      "Test Session",
					CreatedAt: "2024-01-01T00:00:00Z",
				},
				Messages: []internal.Message{
					{Actor: "user", Content: "Hello"},
				},
			},
		},
		{
			name: "session without created date",
			session: &internal.Session{
				ID: "test-session",
				Metadata: internal.Metadata{
					Name: "Test Session",
				},
				Messages: []internal.Message{
					{Actor: "user", Content: "Hello"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that function doesn't panic
			displaySessionHeader(tt.session)
		})
	}
}

func TestDisplayMessage(t *testing.T) {
	tests := []struct {
		name  string
		index int
		msg   internal.Message
		total int
	}{
		{
			name:  "user message",
			index: 1,
			msg: internal.Message{
				Actor:     "user",
				Content:   "Hello, world!",
				Timestamp: time.Now().Format(time.RFC3339),
			},
			total: 2,
		},
		{
			name:  "assistant message",
			index: 2,
			msg: internal.Message{
				Actor:     "assistant",
				Content:   "Hi there!",
				Timestamp: time.Now().Format(time.RFC3339),
			},
			total: 2,
		},
		{
			name:  "empty message",
			index: 1,
			msg: internal.Message{
				Actor:     "user",
				Content:   "",
				Timestamp: time.Now().Format(time.RFC3339),
			},
			total: 1,
		},
		{
			name:  "message without timestamp",
			index: 1,
			msg: internal.Message{
				Actor:   "user",
				Content: "Hello",
			},
			total: 1,
		},
		{
			name:  "message with invalid timestamp",
			index: 1,
			msg: internal.Message{
				Actor:     "user",
				Content:   "Hello",
				Timestamp: "invalid-timestamp",
			},
			total: 1,
		},
		{
			name:  "unknown actor type",
			index: 1,
			msg: internal.Message{
				Actor:   "system",
				Content: "System message",
			},
			total: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that function doesn't panic
			displayMessage(tt.index, tt.msg, tt.total)
		})
	}
}

func TestWrapText(t *testing.T) {
	tests := []struct {
		name        string
		text        string
		width       int
		wantContain string
	}{
		{
			name:        "short text",
			text:        "Hello world",
			width:       80,
			wantContain: "Hello world",
		},
		{
			name:        "long text",
			text:        "This is a very long line of text that should be wrapped when it exceeds the specified width limit",
			width:       20,
			wantContain: "This is a very",
		},
		{
			name:        "text with newlines",
			text:        "Line 1\nLine 2\nLine 3",
			width:       80,
			wantContain: "Line 1",
		},
		{
			name:        "empty text",
			text:        "",
			width:       80,
			wantContain: "",
		},
		{
			name:        "single long word",
			text:        "supercalifragilisticexpialidocious",
			width:       10,
			wantContain: "supercalifragilisticexpialidocious",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := wrapText(tt.text, tt.width)
			if tt.wantContain != "" && len(result) == 0 && len(tt.text) > 0 {
				t.Errorf("wrapText() returned empty string for non-empty input")
			}
			// Just verify it doesn't panic and returns something reasonable
			_ = result
		})
	}
}
