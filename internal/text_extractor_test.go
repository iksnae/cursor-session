package internal

import (
	"strings"
	"testing"
)

func TestExtractTextFromBubble(t *testing.T) {
	tests := []struct {
		name    string
		bubble  *RawBubble
		want    string
		wantErr bool
	}{
		{
			name: "primary text",
			bubble: &RawBubble{
				Text:       "Hello world",
				RichText:   "",
				CodeBlocks: []CodeBlock{},
			},
			want:    "Hello world",
			wantErr: false,
		},
		{
			name: "with code blocks",
			bubble: &RawBubble{
				Text:     "Here's some code:",
				RichText: "",
				CodeBlocks: []CodeBlock{
					{Language: "go", Content: "package main"},
				},
			},
			want:    "Here's some code:\n\n```go\npackage main\n```",
			wantErr: false,
		},
		{
			name: "with rich text",
			bubble: &RawBubble{
				Text:       "",
				RichText:   `{"root":{"children":[{"type":"text","text":"Rich text content"}]}}`,
				CodeBlocks: []CodeBlock{},
			},
			want:    "Rich text content",
			wantErr: false,
		},
		{
			name: "text and rich text (both included if different)",
			bubble: &RawBubble{
				Text:       "Primary text",
				RichText:   `{"root":{"children":[{"type":"text","text":"Rich text"}]}}`,
				CodeBlocks: []CodeBlock{},
			},
			// Implementation adds richText if it's not contained in primary text
			want:    "Primary text\n\nRich text",
			wantErr: false,
		},
		{
			name: "multiple code blocks",
			bubble: &RawBubble{
				Text:     "Code examples:",
				RichText: "",
				CodeBlocks: []CodeBlock{
					{Language: "go", Content: "package main"},
					{Language: "python", Content: "print('hello')"},
				},
			},
			want:    "Code examples:\n\n```go\npackage main\n```\n\n```python\nprint('hello')\n```",
			wantErr: false,
		},
		{
			name: "code block without language",
			bubble: &RawBubble{
				Text:     "Code:",
				RichText: "",
				CodeBlocks: []CodeBlock{
					{Content: "just code"},
				},
			},
			want:    "Code:\n\n```\njust code\n```",
			wantErr: false,
		},
		{
			name: "empty bubble",
			bubble: &RawBubble{
				Text:       "",
				RichText:   "",
				CodeBlocks: []CodeBlock{},
			},
			// Implementation returns placeholder for empty bubbles
			want:    "[Message with no extractable text content]",
			wantErr: false,
		},
		{
			name: "rich text with fallback extraction",
			bubble: &RawBubble{
				Text:       "",
				RichText:   `{"invalid": json}`,
				CodeBlocks: []CodeBlock{},
			},
			// Should use fallback extraction
			want:    "[Message with no extractable text content]",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractTextFromBubble(tt.bubble)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractTextFromBubble() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ExtractTextFromBubble() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractFallbackText(t *testing.T) {
	tests := []struct {
		name    string
		jsonStr string
		want    string
	}{
		{
			name:    "simple text field",
			jsonStr: `{"text": "Hello world"}`,
			want:    "Hello world ",
		},
		{
			name:    "multiple text fields",
			jsonStr: `{"text": "First", "other": "ignored", "text": "Second"}`,
			want:    "First Second ",
		},
		{
			name:    "escaped quotes",
			jsonStr: `{"text": "Hello \"world\""}`,
			// The function doesn't unescape, so it extracts the raw string including escapes
			want:    "Hello \\\"world\\\" ",
		},
		{
			name:    "no text field",
			jsonStr: `{"other": "value"}`,
			want:    "",
		},
		{
			name:    "empty string",
			jsonStr: "",
			want:    "",
		},
		{
			name:    "text with spaces around colon",
			jsonStr: `{"text" : "value"}`,
			want:    "value ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractFallbackText(tt.jsonStr)
			// Trim space for comparison since the function adds trailing space
			got = strings.TrimSpace(got)
			want := strings.TrimSpace(tt.want)
			if got != want {
				t.Errorf("extractFallbackText() = %q, want %q", got, want)
			}
		})
	}
}

func TestExtractFromRawJSON(t *testing.T) {
	tests := []struct {
		name    string
		jsonStr string
		want    string
	}{
		{
			name:    "content field",
			jsonStr: `{"content": "Hello world"}`,
			want:    "Hello world",
		},
		{
			name:    "value field",
			jsonStr: `{"value": "This is a longer test value that exceeds minimum length"}`,
			want:    "This is a longer test value that exceeds minimum length",
		},
		{
			name:    "message field",
			jsonStr: `{"message": "Test message"}`,
			want:    "Test message",
		},
		{
			name:    "thinking field",
			jsonStr: `{"thinking": "Some thinking text"}`,
			want:    "Some thinking text",
		},
		{
			name:    "multiple fields",
			jsonStr: `{"content": "First content here", "value": "Second value here"}`,
			want:    "First content here\nSecond value here",
		},
		{
			name:    "escaped characters",
			jsonStr: `{"content": "Hello\\nWorld"}`,
			want:    "Hello\nWorld",
		},
		{
			name:    "short value (ignored)",
			jsonStr: `{"content": "short"}`,
			want:    "",
		},
		{
			name:    "no matching fields",
			jsonStr: `{"other": "value"}`,
			want:    "",
		},
		{
			name:    "empty string",
			jsonStr: "",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractFromRawJSON(tt.jsonStr)
			if got != tt.want {
				t.Errorf("extractFromRawJSON() = %q, want %q", got, tt.want)
			}
		})
	}
}
