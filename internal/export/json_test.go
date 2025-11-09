package export

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/iksnae/cursor-session/internal"
)

func TestJSONExporter_Export(t *testing.T) {
	tests := []struct {
		name    string
		session *internal.Session
		wantErr bool
	}{
		{
			name:    "basic session",
			session: internal.CreateTestSession("test1"),
			wantErr: false,
		},
		{
			name:    "empty session",
			session: internal.CreateTestSessionWithMessages("test2", []internal.Message{}),
			wantErr: false,
		},
		{
			name: "session with all fields",
			session: &internal.Session{
				ID:        "test3",
				Workspace: "workspace1",
				Source:    "globalStorage",
				Messages: []internal.Message{
					{
						Actor:     "user",
						Content:   "Hello",
						Timestamp: "2023-01-01T00:00:00Z",
					},
				},
				Metadata: internal.Metadata{
					Name:         "Test",
					ComposerID:   "composer1",
					MessageCount: 1,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			exporter := &JSONExporter{}

			err := exporter.Export(tt.session, &buf)
			if (err != nil) != tt.wantErr {
				t.Errorf("JSONExporter.Export() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				output := buf.String()
				// Verify it's valid JSON
				var session internal.Session
				if err := json.Unmarshal([]byte(output), &session); err != nil {
					t.Errorf("Output is not valid JSON: %v\nOutput: %s", err, output)
					return
				}

				// Verify session ID is present
				if !strings.Contains(output, tt.session.ID) {
					t.Errorf("Output should contain session ID %q", tt.session.ID)
				}

				// Verify it's pretty-printed (contains indentation)
				if !strings.Contains(output, "  ") {
					t.Errorf("Output should be pretty-printed with indentation")
				}
			}
		})
	}
}

func TestJSONExporter_Extension(t *testing.T) {
	exporter := &JSONExporter{}
	if got := exporter.Extension(); got != "json" {
		t.Errorf("JSONExporter.Extension() = %v, want json", got)
	}
}
