package export

import (
	"testing"
)

func TestNewExporter(t *testing.T) {
	tests := []struct {
		name     string
		format   string
		wantType string
		wantExt  string
		wantErr  bool
	}{
		{
			name:     "jsonl format",
			format:   "jsonl",
			wantType: "JSONLExporter",
			wantExt:  "jsonl",
			wantErr:  false,
		},
		{
			name:     "markdown format",
			format:   "md",
			wantType: "MarkdownExporter",
			wantExt:  "md",
			wantErr:  false,
		},
		{
			name:     "markdown format long",
			format:   "markdown",
			wantType: "MarkdownExporter",
			wantExt:  "md",
			wantErr:  false,
		},
		{
			name:     "yaml format",
			format:   "yaml",
			wantType: "YAMLExporter",
			wantExt:  "yaml",
			wantErr:  false,
		},
		{
			name:     "json format",
			format:   "json",
			wantType: "JSONExporter",
			wantExt:  "json",
			wantErr:  false,
		},
		{
			name:     "unsupported format",
			format:   "xml",
			wantType: "",
			wantExt:  "",
			wantErr:  true,
		},
		{
			name:     "empty format",
			format:   "",
			wantType: "",
			wantExt:  "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exporter, err := NewExporter(tt.format)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewExporter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if exporter == nil {
					t.Error("NewExporter() returned nil exporter")
					return
				}

				// Verify extension
				if got := exporter.Extension(); got != tt.wantExt {
					t.Errorf("Exporter.Extension() = %v, want %v", got, tt.wantExt)
				}

				// Verify type (rough check)
				switch tt.wantType {
				case "JSONLExporter":
					if _, ok := exporter.(*JSONLExporter); !ok {
						t.Errorf("Expected JSONLExporter, got %T", exporter)
					}
				case "MarkdownExporter":
					if _, ok := exporter.(*MarkdownExporter); !ok {
						t.Errorf("Expected MarkdownExporter, got %T", exporter)
					}
				case "YAMLExporter":
					if _, ok := exporter.(*YAMLExporter); !ok {
						t.Errorf("Expected YAMLExporter, got %T", exporter)
					}
				case "JSONExporter":
					if _, ok := exporter.(*JSONExporter); !ok {
						t.Errorf("Expected JSONExporter, got %T", exporter)
					}
				}
			} else {
				if exporter != nil {
					t.Errorf("NewExporter() returned exporter %T, want nil", exporter)
				}
			}
		})
	}
}
