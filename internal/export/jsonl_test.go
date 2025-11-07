package export

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/iksnae/cursor-session/internal"
)

func TestJSONLExporter_Export(t *testing.T) {
	tests := []struct {
		name    string
		session *internal.Session
		want    []string
		wantErr bool
	}{
		{
			name:    "empty session",
			session: internal.CreateTestSessionWithMessages("test1", []internal.Message{}),
			want:    []string{}, // No messages means no output lines
			wantErr: false,
		},
		{
			name:    "session with messages",
			session: internal.CreateTestSession("test2"),
			want: []string{
				`"actor":"user"`,
				`"actor":"assistant"`,
			},
			wantErr: false,
		},
		{
			name: "session with timestamp",
			session: internal.CreateTestSessionWithMessages("test3", []internal.Message{
				{
					Actor:     "user",
					Content:   "Hello",
					Timestamp: "2023-01-01T00:00:00Z",
				},
			}),
			want: []string{
				`"timestamp":"2023-01-01T00:00:00Z"`,
			},
			wantErr: false,
		},
		{
			name: "session without timestamp",
			session: internal.CreateTestSessionWithMessages("test4", []internal.Message{
				{
					Actor:   "user",
					Content: "Hello",
				},
			}),
			want: []string{
				`"actor":"user"`,
				`"content":"Hello"`,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			exporter := &JSONLExporter{}

			err := exporter.Export(tt.session, &buf)
			if (err != nil) != tt.wantErr {
				t.Errorf("JSONLExporter.Export() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				output := buf.String()
				// For empty sessions, output should be empty
				if len(tt.session.Messages) == 0 && output != "" {
					t.Errorf("Empty session should produce empty output, got: %q", output)
					return
				}
				
				// Verify each line is valid JSON (only if there are messages)
				if len(tt.session.Messages) > 0 {
					lines := strings.Split(strings.TrimSpace(output), "\n")
					for i, line := range lines {
						if line == "" {
							continue // Skip empty lines
						}
						var msg map[string]interface{}
						if err := json.Unmarshal([]byte(line), &msg); err != nil {
							t.Errorf("Line %d is not valid JSON: %v", i, err)
						}
						// Verify required fields
						if _, ok := msg["actor"]; !ok {
							t.Errorf("Line %d missing 'actor' field", i)
						}
						if _, ok := msg["content"]; !ok {
							t.Errorf("Line %d missing 'content' field", i)
						}
					}

					// Verify expected content
					for _, wantStr := range tt.want {
						if !strings.Contains(output, wantStr) {
							t.Errorf("Output should contain %q", wantStr)
						}
					}
				}
			}
		})
	}
}

func TestJSONLExporter_Extension(t *testing.T) {
	exporter := &JSONLExporter{}
	if got := exporter.Extension(); got != "jsonl" {
		t.Errorf("JSONLExporter.Extension() = %v, want jsonl", got)
	}
}

func TestJSONLExporter_Export_NilSession(t *testing.T) {
	var buf bytes.Buffer
	exporter := &JSONLExporter{}

	// The current implementation will panic on nil, so we test that it does
	defer func() {
		if r := recover(); r == nil {
			t.Error("Export() should panic on nil session")
		}
	}()
	exporter.Export(nil, &buf)
}


