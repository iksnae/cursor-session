package export

import (
	"bytes"
	"strings"
	"testing"

	"github.com/iksnae/cursor-session/internal"
	"gopkg.in/yaml.v3"
)

func TestYAMLExporter_Export(t *testing.T) {
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
			exporter := &YAMLExporter{}

			err := exporter.Export(tt.session, &buf)
			if (err != nil) != tt.wantErr {
				t.Errorf("YAMLExporter.Export() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				output := buf.String()
				// Verify it's valid YAML
				var session internal.Session
				if err := yaml.Unmarshal([]byte(output), &session); err != nil {
					t.Errorf("Output is not valid YAML: %v\nOutput: %s", err, output)
					return
				}

				// Verify session ID is present
				if !strings.Contains(output, tt.session.ID) {
					t.Errorf("Output should contain session ID %q", tt.session.ID)
				}
			}
		})
	}
}

func TestYAMLExporter_Extension(t *testing.T) {
	exporter := &YAMLExporter{}
	if got := exporter.Extension(); got != "yaml" {
		t.Errorf("YAMLExporter.Extension() = %v, want yaml", got)
	}
}
