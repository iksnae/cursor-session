# Export Format Specifications

## Overview

This document specifies the export formats for Cursor chat sessions: JSONL, Markdown, and HTML.

## 1. JSONL Format

### Specification

**Format**: One JSON object per line (JSON Lines)
**Encoding**: UTF-8
**Line Endings**: Unix-style (`\n`)

### Schema

Each line is a complete JSON object representing a single message:

```json
{"timestamp": "2025-11-07T14:23:01Z", "actor": "user", "content": "What is Cursor?"}
{"timestamp": "2025-11-07T14:23:03Z", "actor": "assistant", "content": "Cursor is an AI coding editor..."}
```

### Field Ordering

Fields should appear in this order for consistency:

1. `timestamp` (if present)
2. `actor` (required)
3. `content` (required)

### Encoding Rules

- **Special Characters**: Escape according to JSON spec
- **Newlines**: Preserve in content, escape as `\n`
- **Unicode**: Preserve as-is (UTF-8)
- **Control Characters**: Escape (e.g., `\t`, `\r`)

### Implementation

```go
func ExportJSONL(session Session, w io.Writer) error {
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
```

### Example Output

```jsonl
{"timestamp":"2025-11-07T14:23:01Z","actor":"user","content":"What is Cursor?"}
{"timestamp":"2025-11-07T14:23:03Z","actor":"assistant","content":"Cursor is an AI coding editor that uses AI to help you code."}
{"timestamp":"2025-11-07T14:23:10Z","actor":"user","content":"How do I use it?"}
{"timestamp":"2025-11-07T14:23:12Z","actor":"assistant","content":"You can use Cursor by:\n\n1. Opening a file\n2. Asking questions in the chat\n3. Using AI suggestions"}
```

### File Naming

Format: `session_{id}.jsonl` or `session_{timestamp}.jsonl`

Example: `session_2025-11-07T14-23-01Z.jsonl`

## 2. Markdown Format

### Specification

**Format**: Markdown document
**Encoding**: UTF-8
**Line Endings**: Unix-style (`\n`)

### Structure

```markdown
# Session {session-id}

**Workspace:** {workspace-path}  
**Source:** {source}  
**Messages:** {count}

---

## Messages

**user:** {content}

**assistant:** {content}

**user:** {content}

**assistant:** {content}
```

### Header Section

- Session ID as main heading
- Metadata as list items
- Horizontal rule separator

### Message Formatting

- **User messages**: `**user:** {content}`
- **Assistant messages**: `**assistant:** {content}`
- **Tool messages**: `**tool:** {content}`

### Content Handling

- **Code blocks**: Preserve markdown code fences
- **Inline code**: Preserve backticks
- **Multi-line**: Preserve line breaks
- **Special characters**: Escape markdown syntax if needed

### Timestamp Display

Optional timestamp in parentheses:

```markdown
**user:** (2025-11-07T14:23:01Z) What is Cursor?
```

### Implementation

```go
func ExportMarkdown(session Session, w io.Writer) error {
    // Header
    fmt.Fprintf(w, "# Session %s\n\n", session.ID)

    if session.Workspace != "" {
        fmt.Fprintf(w, "**Workspace:** %s  \n", session.Workspace)
    }
    fmt.Fprintf(w, "**Source:** %s  \n", session.Source)
    fmt.Fprintf(w, "**Messages:** %d\n\n", len(session.Messages))
    fmt.Fprintf(w, "---\n\n")
    fmt.Fprintf(w, "## Messages\n\n")

    // Messages
    for _, msg := range session.Messages {
        timestamp := ""
        if msg.Timestamp != "" {
            timestamp = fmt.Sprintf("(%s) ", msg.Timestamp)
        }

        // Escape markdown in content if needed
        content := escapeMarkdown(msg.Content)

        fmt.Fprintf(w, "**%s:** %s%s\n\n", msg.Actor, timestamp, content)
    }

    return nil
}

func escapeMarkdown(text string) string {
    // Basic escaping - may need more sophisticated handling
    text = strings.ReplaceAll(text, "**", "\\*\\*")
    text = strings.ReplaceAll(text, "__", "\\_\\_")
    return text
}
```

### Example Output

```markdown
# Session session_2025-11-07_a1b2c3d4

**Workspace:** /Users/k/Projects/my-app  
**Source:** sqlite  
**Messages:** 4

---

## Messages

**user:** (2025-11-07T14:23:01Z) What is Cursor?

**assistant:** (2025-11-07T14:23:03Z) Cursor is an AI coding editor that uses AI to help you code.

**user:** (2025-11-07T14:23:10Z) How do I use it?

**assistant:** (2025-11-07T14:23:12Z) You can use Cursor by:

1. Opening a file
2. Asking questions in the chat
3. Using AI suggestions
```

### File Naming

Format: `session_{id}.md` or `session_{timestamp}.md`

Example: `session_2025-11-07T14-23-01Z.md`

## 3. HTML Format

### Specification

**Format**: Self-contained HTML document
**Encoding**: UTF-8
**Styling**: Embedded CSS

### Structure

```html
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>Session {id}</title>
    <style>
      /* Embedded CSS */
    </style>
  </head>
  <body>
    <header>
      <h1>Session {id}</h1>
      <div class="metadata">...</div>
    </header>
    <main>
      <div class="messages">
        <div class="message user">...</div>
        <div class="message assistant">...</div>
      </div>
    </main>
  </body>
</html>
```

### Styling

```css
body {
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
  max-width: 800px;
  margin: 0 auto;
  padding: 20px;
  line-height: 1.6;
}

.message {
  margin: 1em 0;
  padding: 1em;
  border-radius: 8px;
}

.message.user {
  background-color: #e3f2fd;
  margin-left: 20%;
}

.message.assistant {
  background-color: #f5f5f5;
  margin-right: 20%;
}

.message.tool {
  background-color: #fff3e0;
  font-style: italic;
}

.timestamp {
  font-size: 0.85em;
  color: #666;
  margin-bottom: 0.5em;
}
```

### Message Formatting

```html
<div class="message user">
  <div class="timestamp">2025-11-07T14:23:01Z</div>
  <div class="content">What is Cursor?</div>
</div>
```

### Code Block Handling

Preserve code blocks with syntax highlighting:

```html
<pre><code class="language-go">package main

func main() {
    // code here
}
</code></pre>
```

### Implementation

```go
func ExportHTML(session Session, w io.Writer) error {
    // Write HTML template
    tmpl := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Session {{.ID}}</title>
    <style>
        /* CSS here */
    </style>
</head>
<body>
    <header>
        <h1>Session {{.ID}}</h1>
    </header>
    <main>
        {{range .Messages}}
        <div class="message {{.Actor}}">
            {{if .Timestamp}}
            <div class="timestamp">{{.Timestamp}}</div>
            {{end}}
            <div class="content">{{.Content}}</div>
        </div>
        {{end}}
    </main>
</body>
</html>`

    t := template.Must(template.New("html").Parse(tmpl))
    return t.Execute(w, session)
}
```

### Example Output

See `examples/session.html` for full example.

### File Naming

Format: `session_{id}.html` or `session_{timestamp}.html`

Example: `session_2025-11-07T14-23-01Z.html`

## 4. Export Function Interface

### Common Interface

```go
type Exporter interface {
    Export(session Session, w io.Writer) error
}

type JSONLExporter struct{}
type MarkdownExporter struct{}
type HTMLExporter struct{}

func (e *JSONLExporter) Export(session Session, w io.Writer) error {
    // Implementation
}

func (e *MarkdownExporter) Export(session Session, w io.Writer) error {
    // Implementation
}

func (e *HTMLExporter) Export(session Session, w io.Writer) error {
    // Implementation
}
```

### Factory Function

```go
func NewExporter(format string) (Exporter, error) {
    switch format {
    case "jsonl":
        return &JSONLExporter{}, nil
    case "md", "markdown":
        return &MarkdownExporter{}, nil
    case "html":
        return &HTMLExporter{}, nil
    default:
        return nil, fmt.Errorf("unsupported format: %s", format)
    }
}
```

## 5. Streaming Export

### Memory Efficiency

For large sessions, use streaming:

```go
func ExportStreaming(sessions []Session, format string, outputDir string) error {
    exporter, err := NewExporter(format)
    if err != nil {
        return err
    }

    for _, session := range sessions {
        filename := fmt.Sprintf("session_%s.%s", session.ID, getExtension(format))
        filepath := filepath.Join(outputDir, filename)

        file, err := os.Create(filepath)
        if err != nil {
            return fmt.Errorf("failed to create file: %w", err)
        }

        if err := exporter.Export(session, file); err != nil {
            file.Close()
            return err
        }

        file.Close()
    }

    return nil
}
```

## 6. Format Comparison

| Format   | Pros                                     | Cons               | Use Case              |
| -------- | ---------------------------------------- | ------------------ | --------------------- |
| JSONL    | Machine-readable, streaming, compact     | Not human-readable | Integration, analysis |
| Markdown | Human-readable, version-control friendly | Less structured    | Documentation, review |
| HTML     | Rich formatting, interactive             | Larger file size   | Presentation, sharing |

## 7. Next Steps

1. Implement export functions
2. Create example outputs
3. Test with real session data
4. Validate format correctness
5. Performance test with large sessions
