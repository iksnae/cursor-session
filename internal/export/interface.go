package export

import (
	"fmt"
	"io"

	"github.com/iksnae/cursor-session/internal"
)

// Exporter defines the interface for all export formats
type Exporter interface {
	Export(session *internal.Session, w io.Writer) error
	Extension() string
}

// NewExporter creates a new exporter based on format
func NewExporter(format string) (Exporter, error) {
	switch format {
	case "jsonl":
		return &JSONLExporter{}, nil
	case "md", "markdown":
		return &MarkdownExporter{}, nil
	case "yaml":
		return &YAMLExporter{}, nil
	case "json":
		return &JSONExporter{}, nil
	default:
		return nil, fmt.Errorf("unsupported format: %s (supported: jsonl, md, yaml, json)", format)
	}
}
