package export

import (
	"encoding/json"
	"io"

	"github.com/iksnae/cursor-session/internal"
)

// JSONExporter exports sessions in JSON format (pretty-printed)
type JSONExporter struct{}

// Export exports a session to JSON format
func (e *JSONExporter) Export(session *internal.Session, w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")

	return enc.Encode(session)
}

// Extension returns the file extension for this format
func (e *JSONExporter) Extension() string {
	return "json"
}
