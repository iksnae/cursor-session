package export

import (
	"fmt"
	"io"
	"strings"

	"github.com/iksnae/cursor-session/internal"
)

// MarkdownExporter exports sessions in Markdown format
type MarkdownExporter struct{}

// Export exports a session to Markdown format
func (e *MarkdownExporter) Export(session *internal.Session, w io.Writer) error {
	// Header
	_, _ = fmt.Fprintf(w, "# Session %s\n\n", session.ID)

	if session.Workspace != "" {
		_, _ = fmt.Fprintf(w, "**Workspace:** %s  \n", session.Workspace)
	}
	_, _ = fmt.Fprintf(w, "**Source:** %s  \n", session.Source)
	_, _ = fmt.Fprintf(w, "**Messages:** %d\n\n", len(session.Messages))

	if session.Metadata.Name != "" {
		_, _ = fmt.Fprintf(w, "**Name:** %s\n\n", session.Metadata.Name)
	}

	_, _ = fmt.Fprintf(w, "---\n\n")
	_, _ = fmt.Fprintf(w, "## Messages\n\n")

	// Messages
	for i, msg := range session.Messages {
		timestamp := ""
		if msg.Timestamp != "" {
			timestamp = fmt.Sprintf(" (%s)", msg.Timestamp)
		}

		// Escape markdown in content if needed
		content := escapeMarkdown(msg.Content)

		_, _ = fmt.Fprintf(w, "**%s:**%s\n\n%s\n\n", msg.Actor, timestamp, content)

		// Add horizontal rule after each message (except the last one)
		if i < len(session.Messages)-1 {
			_, _ = fmt.Fprintf(w, "---\n\n")
		}
	}

	return nil
}

// escapeMarkdown escapes markdown special characters
func escapeMarkdown(text string) string {
	// Basic escaping - preserve code blocks
	lines := strings.Split(text, "\n")
	var result []string
	inCodeBlock := false

	for _, line := range lines {
		if strings.HasPrefix(line, "```") {
			inCodeBlock = !inCodeBlock
			result = append(result, line)
		} else if inCodeBlock {
			result = append(result, line)
		} else {
			// Escape markdown syntax outside code blocks
			line = strings.ReplaceAll(line, "**", "\\*\\*")
			line = strings.ReplaceAll(line, "__", "\\_\\_")
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}

// Extension returns the file extension for this format
func (e *MarkdownExporter) Extension() string {
	return "md"
}
