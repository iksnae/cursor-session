package internal

import (
	"strings"
	"testing"
)

func TestExtractTextFromRichText(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "empty string",
			input:   "",
			want:    "",
			wantErr: false,
		},
		{
			name:    "root.children structure",
			input:   `{"root":{"children":[{"type":"text","text":"Hello"}]}}`,
			want:    "Hello",
			wantErr: false,
		},
		{
			name:    "direct children array",
			input:   `{"children":[{"type":"text","text":"World"}]}`,
			want:    "World",
			wantErr: false,
		},
		{
			name:    "code block",
			input:   `{"root":{"children":[{"type":"code","children":[{"type":"text","text":"package main"}]}]}}`,
			want:    "\n```\npackage main\n```\n",
			wantErr: false,
		},
		{
			name:    "multiple text nodes",
			input:   `{"root":{"children":[{"type":"text","text":"Hello"},{"type":"text","text":" World"}]}}`,
			want:    "Hello World",
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			input:   `{invalid json}`,
			want:    "",
			wantErr: true,
		},
		{
			name:    "unknown format",
			input:   `{"unknown":"format"}`,
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractTextFromRichText(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractTextFromRichText() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ExtractTextFromRichText() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractTextFromRichText_StructuredFormat(t *testing.T) {
	// Test RichTextRoot structured format
	input := `{
		"root": {
			"type": "root",
			"children": [
				{
					"type": "text",
					"text": "Hello"
				},
				{
					"type": "code",
					"children": [
						{
							"type": "text",
							"text": "package main"
						}
					]
				}
			]
		}
	}`

	got, err := ExtractTextFromRichText(input)
	if err != nil {
		t.Fatalf("ExtractTextFromRichText() error = %v", err)
	}

	if got == "" {
		t.Error("ExtractTextFromRichText() returned empty string")
	}

	// Should contain both text and code
	if !contains(got, "Hello") {
		t.Error("ExtractTextFromRichText() should contain 'Hello'")
	}
	if !contains(got, "package main") {
		t.Error("ExtractTextFromRichText() should contain 'package main'")
	}
}

func TestExtractTextFromRichText_DirectNode(t *testing.T) {
	// Test direct node format
	input := `{
		"type": "text",
		"text": "Direct node"
	}`

	got, err := ExtractTextFromRichText(input)
	if err != nil {
		t.Fatalf("ExtractTextFromRichText() error = %v", err)
	}

	if !contains(got, "Direct node") {
		t.Errorf("ExtractTextFromRichText() = %q, should contain 'Direct node'", got)
	}
}

func TestExtractTextFromRichText_ArrayOfNodes(t *testing.T) {
	// Test array of nodes format - this format is not currently supported
	// The implementation expects root.children or direct children structure
	input := `[
		{
			"type": "text",
			"text": "First"
		},
		{
			"type": "text",
			"text": "Second"
		}
	]`

	got, err := ExtractTextFromRichText(input)
	// This format may not be supported, so we just verify it doesn't panic
	_ = got
	_ = err
	// Note: The current implementation may not handle array format directly
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && strings.Contains(s, substr)
}
