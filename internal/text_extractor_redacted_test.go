package internal

import (
	"strings"
	"testing"
)

func TestReformatRedactedReasoning(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		want     string
		wantCode bool // Whether output should contain code block markers
	}{
		{
			name:     "only redacted reasoning",
			input:    "[Redacted Reasoning: gGWXW9zGta5OUCt7uy5c91odL2W4LjbXfer65wl1l4rC23q1Km]",
			wantCode: true,
		},
		{
			name:     "redacted reasoning with whitespace",
			input:    "  [Redacted Reasoning: hash123]  ",
			wantCode: true,
		},
		{
			name:     "text with redacted reasoning",
			input:    "Some text [Redacted Reasoning: hash123] more text",
			wantCode: true,
		},
		{
			name:     "normal text without redacted reasoning",
			input:    "This is normal text",
			want:     "This is normal text",
			wantCode: false,
		},
		{
			name:     "empty string",
			input:    "",
			want:     "",
			wantCode: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reformatRedactedReasoning(tt.input)

			if tt.want != "" {
				if got != tt.want {
					t.Errorf("reformatRedactedReasoning() = %q, want %q", got, tt.want)
				}
			}

			if tt.wantCode {
				// Should contain code block markers
				if !strings.Contains(got, "```") {
					t.Errorf("reformatRedactedReasoning() should contain code block markers, got %q", got)
				}
				if !strings.Contains(got, "[Redacted Reasoning]") {
					t.Errorf("reformatRedactedReasoning() should contain [Redacted Reasoning], got %q", got)
				}
			} else {
				// Should not contain code block markers
				if strings.Contains(got, "```") {
					t.Errorf("reformatRedactedReasoning() should not contain code block markers, got %q", got)
				}
			}
		})
	}
}
