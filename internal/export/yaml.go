package export

import (
	"io"

	"github.com/iksnae/cursor-session/internal"
	"gopkg.in/yaml.v3"
)

// YAMLExporter exports sessions in YAML format
type YAMLExporter struct{}

// Export exports a session to YAML format
func (e *YAMLExporter) Export(session *internal.Session, w io.Writer) error {
	enc := yaml.NewEncoder(w)
	defer func() { _ = enc.Close() }()

	return enc.Encode(session)
}

// Extension returns the file extension for this format
func (e *YAMLExporter) Extension() string {
	return "yaml"
}
