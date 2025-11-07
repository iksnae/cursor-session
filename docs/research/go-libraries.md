# Go Library Research & Evaluation

## Overview

This document evaluates Go libraries for implementing the Cursor Session Export CLI, focusing on SQLite access, JSON parsing, CLI framework, and file I/O.

## 1. SQLite Driver: modernc.org/sqlite

### Evaluation

**Package**: `modernc.org/sqlite`

**Key Features**:

- Pure Go implementation (no CGO required)
- Cross-platform compatibility
- Standard `database/sql` interface
- Good performance for read operations

### Advantages

- ✅ No CGO dependency (easier cross-compilation)
- ✅ Works on macOS, Linux, Windows
- ✅ Familiar `database/sql` API
- ✅ Active maintenance
- ✅ Suitable for read-only operations

### Disadvantages

- ⚠️ Slightly slower than CGO-based drivers (acceptable for this use case)
- ⚠️ Larger binary size than CGO alternatives

### Usage Pattern

```go
import (
    "database/sql"
    _ "modernc.org/sqlite"
)

func OpenDatabase(path string) (*sql.DB, error) {
    db, err := sql.Open("sqlite", path)
    if err != nil {
        return nil, fmt.Errorf("failed to open database: %w", err)
    }

    // Test connection
    if err := db.Ping(); err != nil {
        return nil, fmt.Errorf("database ping failed: %w", err)
    }

    return db, nil
}

func QuerySessions(db *sql.DB) ([]Session, error) {
    query := `
        SELECT key, value
        FROM ItemTable
        WHERE key IN (?, ?, ?)
    `

    rows, err := db.Query(query,
        "aiService.prompts",
        "workbench.panel.aichat.view.aichat.chatdata",
        "composerData")
    if err != nil {
        return nil, fmt.Errorf("query failed: %w", err)
    }
    defer rows.Close()

    var sessions []Session
    for rows.Next() {
        var key, value string
        if err := rows.Scan(&key, &value); err != nil {
            continue // Skip corrupt rows
        }
        // Parse value as JSON...
    }

    return sessions, rows.Err()
}
```

### Performance Considerations

- Read-only operations are fast (< 100ms per database)
- Connection pooling not needed for single-threaded scans
- Consider connection reuse for multiple workspace scans

### Error Handling

- Handle `sql.ErrNoRows` gracefully
- Check for database lock errors (Cursor may have DB open)
- Validate database file exists and is readable

## 2. JSON Parsing: github.com/tidwall/gjson

### Evaluation

**Package**: `github.com/tidwall/gjson`

**Key Features**:

- Fast path-based JSON extraction
- No need to unmarshal entire JSON
- Path query syntax
- Good for extracting nested fields

### Advantages

- ✅ Very fast for path-based queries
- ✅ Memory efficient (doesn't parse entire JSON)
- ✅ Simple API for extracting specific fields
- ✅ Handles malformed JSON gracefully

### Comparison with encoding/json

| Feature  | gjson                   | encoding/json        |
| -------- | ----------------------- | -------------------- |
| Speed    | Faster for path queries | Slower (full parse)  |
| Memory   | Lower (streaming)       | Higher (full object) |
| API      | Path-based              | Struct-based         |
| Use Case | Extract specific fields | Full object parsing  |

### Usage Pattern

```go
import "github.com/tidwall/gjson"

func ExtractMessages(jsonStr string) []Message {
    var messages []Message

    // Extract messages array
    messagesJSON := gjson.Get(jsonStr, "messages")
    if !messagesJSON.Exists() {
        return messages
    }

    // Iterate over messages
    messagesJSON.ForEach(func(key, value gjson.Result) bool {
        role := value.Get("role").String()
        content := value.Get("content").String()
        timestamp := value.Get("timestamp").String()

        messages = append(messages, Message{
            Actor:     role,
            Content:   content,
            Timestamp: timestamp,
        })
        return true // continue iteration
    })

    return messages
}
```

### Recommendation

- Use `gjson` for extracting messages from SQLite values
- Use `encoding/json` for final export serialization
- Best of both worlds: fast extraction + standard serialization

## 3. CLI Framework: github.com/spf13/cobra

### Evaluation

**Package**: `github.com/spf13/cobra`

**Key Features**:

- Industry-standard CLI framework
- Command/subcommand structure
- Flag and argument handling
- Auto-generated help text
- Shell completion support

### Advantages

- ✅ Mature and widely used
- ✅ Excellent documentation
- ✅ Rich feature set
- ✅ Good error handling
- ✅ Supports nested commands

### Command Structure

```go
// cmd/root.go
var rootCmd = &cobra.Command{
    Use:   "cursor-session",
    Short: "Export Cursor Editor chat sessions",
    Long:  "A CLI tool to extract and export chat sessions from Cursor Editor",
}

// cmd/list.go
var listCmd = &cobra.Command{
    Use:   "list",
    Short: "List available sessions",
    RunE: func(cmd *cobra.Command, args []string) error {
        // Implementation
    },
}

// cmd/export.go
var exportCmd = &cobra.Command{
    Use:   "export",
    Short: "Export sessions to file",
    RunE: func(cmd *cobra.Command, args []string) error {
        format, _ := cmd.Flags().GetString("format")
        output, _ := cmd.Flags().GetString("out")
        // Implementation
    },
}

func init() {
    exportCmd.Flags().StringP("format", "f", "jsonl", "Export format (jsonl, md, html)")
    exportCmd.Flags().StringP("out", "o", "./exports", "Output directory")
    exportCmd.Flags().Bool("deep", false, "Deep scan CacheStorage")
    exportCmd.Flags().BoolP("verbose", "v", false, "Verbose output")

    rootCmd.AddCommand(listCmd, exportCmd, scanCmd)
}
```

### Flag Patterns

```go
// String flags
--format, -f: jsonl | md | html
--out, -o: output directory path

// Boolean flags
--deep: enable CacheStorage scanning
--verbose, -v: enable debug logging

// Future flags
--since: filter by timestamp
--workspace: filter by workspace
--contains: search content
```

## 4. File I/O: Standard Library

### Path Handling

**Package**: `path/filepath`

```go
import (
    "path/filepath"
    "os"
    "runtime"
)

// Cross-platform path joining
basePath := filepath.Join(home, "Library/Application Support/Cursor/User")

// Path expansion
expandedPath := os.ExpandEnv("~/.config/Cursor/User") // Not reliable
// Better: use os.UserHomeDir()
home, _ := os.UserHomeDir()
path := filepath.Join(home, ".config/Cursor/User")
```

### Directory Walking

**Package**: `filepath.Walk` or `filepath.WalkDir`

```go
func ScanWorkspaceStorage(root string) ([]string, error) {
    var dbFiles []string

    err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
        if err != nil {
            return err // Continue on error, or return to stop
        }

        if !d.IsDir() && filepath.Base(path) == "state.vscdb" {
            dbFiles = append(dbFiles, path)
        }

        return nil
    })

    return dbFiles, err
}
```

### Binary File Reading

```go
func ReadCacheFile(path string) ([]byte, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("failed to read cache file: %w", err)
    }
    return data, nil
}

// For large files, use streaming
func ReadCacheFileStream(path string) (io.Reader, error) {
    file, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    return file, nil // Caller must close
}
```

## 5. Logging: Structured Logging

### Option 1: Standard log Package

```go
import (
    "log"
    "os"
)

var verbose bool

func logInfo(msg string) {
    log.Printf("[INFO] %s", msg)
}

func logVerbose(msg string) {
    if verbose {
        log.Printf("[DEBUG] %s", msg)
    }
}
```

### Option 2: github.com/rs/zerolog

**Package**: `github.com/rs/zerolog`

**Advantages**:

- Structured logging (JSON output)
- Fast performance
- Level-based filtering
- Context fields

```go
import "github.com/rs/zerolog/log"

func initLogger(verbose bool) {
    level := zerolog.InfoLevel
    if verbose {
        level = zerolog.DebugLevel
    }
    zerolog.SetGlobalLevel(level)
}

log.Info().
    Str("path", dbPath).
    Int("sessions", count).
    Msg("Found sessions in database")
```

### Recommendation

- Start with standard `log` package for simplicity
- Consider `zerolog` if structured logging needed for integration

## 6. Error Handling: Standard Patterns

### Error Wrapping

```go
import "fmt"

func OpenDatabase(path string) (*sql.DB, error) {
    db, err := sql.Open("sqlite", path)
    if err != nil {
        return nil, fmt.Errorf("failed to open database %s: %w", path, err)
    }
    return db, nil
}
```

### Custom Error Types

```go
type StorageError struct {
    Path string
    Err  error
}

func (e *StorageError) Error() string {
    return fmt.Sprintf("storage error at %s: %v", e.Path, e.Err)
}

func (e *StorageError) Unwrap() error {
    return e.Err
}
```

### Graceful Degradation

```go
func ScanDatabases(root string) []Session {
    var allSessions []Session

    dbFiles, err := findDatabases(root)
    if err != nil {
        log.Printf("Warning: failed to scan %s: %v", root, err)
        return allSessions // Return empty, don't fail completely
    }

    for _, dbPath := range dbFiles {
        sessions, err := extractSessions(dbPath)
        if err != nil {
            log.Printf("Warning: failed to parse %s: %v", dbPath, err)
            continue // Skip this database, continue with others
        }
        allSessions = append(allSessions, sessions...)
    }

    return allSessions
}
```

## 7. Library Comparison Matrix

| Library            | Purpose            | Performance | Complexity | Recommendation  |
| ------------------ | ------------------ | ----------- | ---------- | --------------- |
| modernc.org/sqlite | SQLite access      | Good        | Low        | ✅ Use          |
| gjson              | JSON parsing       | Excellent   | Low        | ✅ Use          |
| encoding/json      | JSON serialization | Good        | Low        | ✅ Use          |
| cobra              | CLI framework      | N/A         | Medium     | ✅ Use          |
| zerolog            | Logging            | Excellent   | Medium     | ⚠️ Optional     |
| filepath           | Path handling      | N/A         | Low        | ✅ Use (stdlib) |

## 8. Dependencies Summary

```go
// go.mod
module github.com/k/cursor-session

go 1.22

require (
    github.com/spf13/cobra v1.8.0
    github.com/tidwall/gjson v1.17.0
    modernc.org/sqlite v1.28.0
)

require (
    // Transitive dependencies handled by go mod
)
```

## 9. Performance Benchmarks (Estimated)

| Operation         | Estimated Time      | Notes                          |
| ----------------- | ------------------- | ------------------------------ |
| SQLite scan       | < 100ms per DB      | Single query, small result set |
| JSON parsing      | < 10ms per session  | gjson path extraction          |
| CacheStorage scan | 50-200ms per file   | Binary read + heuristic parse  |
| Export (JSONL)    | < 50ms per session  | Stream writing                 |
| Export (Markdown) | < 100ms per session | Template rendering             |

## 10. Next Steps

1. Create test project with all dependencies
2. Benchmark SQLite queries
3. Test JSON parsing performance
4. Validate cross-platform compatibility
5. Create example CLI structure
