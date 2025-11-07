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
