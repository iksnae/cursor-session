package internal

import (
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
