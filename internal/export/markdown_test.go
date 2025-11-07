package export

import (
	"bytes"
	"strings"
	"testing"

	"github.com/iksnae/cursor-session/internal"
)

func TestMarkdownExporter_Export(t *testing.T) {
	tests := []struct {
		name    string
		session *internal.Session
		want    []string
		wantErr bool
	}{
		{
			name:    "basic session",
			session: internal.CreateTestSession("test1"),
			want: []string{
				"# Session test1",
				"**Workspace:** test-workspace",
				"**Source:** globalStorage",
				"**Messages:** 2",
				"## Messages",
				"**user:**",
				"Hello, how are you?",
				"**assistant:**",
			},
			wantErr: false,
		},
		{
			name: "session with timestamp",
			session: internal.CreateTestSessionWithMessages("test2", []internal.Message{
				{
					Actor:     "user",
					Content:   "Hello",
					Timestamp: "2023-01-01T00:00:00Z",
				},
			}),
			want: []string{
				"**user:** (2023-01-01T00:00:00Z)",
			},
			wantErr: false,
		},
		{
			name: "session with name",
			session: &internal.Session{
				ID:        "test3",
				Source:    "globalStorage",
				Messages:  []internal.Message{},
				Metadata: internal.Metadata{
					Name: "My Conversation",
				},
			},
			want: []string{
				"**Name:** My Conversation",
			},
			wantErr: false,
		},
		{
			name: "session without workspace",
			session: &internal.Session{
				ID:       "test4",
				Source:   "globalStorage",
				Messages: []internal.Message{},
			},
			want: []string{
				"# Session test4",
				"**Source:** globalStorage",
			},
			wantErr: false,
		},
		{
			name:    "empty session",
			session: internal.CreateTestSessionWithMessages("test5", []internal.Message{}),
			want: []string{
				"# Session test5",
				"**Messages:** 0",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			exporter := &MarkdownExporter{}

			err := exporter.Export(tt.session, &buf)
			if (err != nil) != tt.wantErr {
				t.Errorf("MarkdownExporter.Export() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				output := buf.String()
				for _, wantStr := range tt.want {
					if !strings.Contains(output, wantStr) {
						t.Errorf("Output should contain %q, got:\n%s", wantStr, output)
					}
				}
			}
		})
	}
}

func TestMarkdownExporter_Extension(t *testing.T) {
	exporter := &MarkdownExporter{}
	if got := exporter.Extension(); got != "md" {
		t.Errorf("MarkdownExporter.Extension() = %v, want md", got)
	}
}

func TestEscapeMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		want     []string
		notWant  []string
	}{
		{
			name:  "basic text",
			input: "Hello world",
			want:  []string{"Hello world"},
		},
		{
			name:    "markdown bold",
			input:    "This is **bold** text",
			want:     []string{"\\*\\*bold\\*\\*"},
			notWant:  []string{"**bold**"},
		},
		{
			name:    "markdown underline",
			input:    "This is __underlined__ text",
			want:     []string{"\\_\\_underlined\\_\\_"},
			notWant:  []string{"__underlined__"},
		},
		{
			name:  "code block preserved",
			input: "```go\npackage main\n```",
			want:  []string{"```go", "package main", "```"},
		},
		{
			name:    "mixed content",
			input:    "Regular text **bold** and ```code```",
			want:     []string{"\\*\\*bold\\*\\*", "```code```"},
			notWant:  []string{"**bold**"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := escapeMarkdown(tt.input)
			for _, wantStr := range tt.want {
				if !strings.Contains(got, wantStr) {
					t.Errorf("escapeMarkdown() should contain %q, got: %s", wantStr, got)
				}
			}
			for _, notWantStr := range tt.notWant {
				if strings.Contains(got, notWantStr) {
					t.Errorf("escapeMarkdown() should not contain %q, got: %s", notWantStr, got)
				}
			}
		})
	}
}


