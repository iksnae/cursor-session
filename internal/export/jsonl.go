package export

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/iksnae/cursor-session/internal"
)

// JSONLExporter exports sessions in JSONL format (one message per line)
type JSONLExporter struct{}

// Export exports a session to JSONL format
func (e *JSONLExporter) Export(session *internal.Session, w io.Writer) error {
	enc := json.NewEncoder(w)

	for _, msg := range session.Messages {
		// Create message object
		obj := map[string]interface{}{
			"actor":   msg.Actor,
			"content": msg.Content,
		}

		// Add timestamp if present
		if msg.Timestamp != "" {
			obj["timestamp"] = msg.Timestamp
		}

		// Encode to single line
		if err := enc.Encode(obj); err != nil {
			return fmt.Errorf("failed to encode message: %w", err)
		}
	}

	return nil
}

// Extension returns the file extension for this format
func (e *JSONLExporter) Extension() string {
	return "jsonl"
}
