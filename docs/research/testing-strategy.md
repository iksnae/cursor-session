# Testing Strategy

## Overview

This document outlines the comprehensive testing approach for the Cursor Session Export CLI, including unit tests, integration tests, and test fixtures.

## 1. Testing Philosophy

### Principles

1. **Test isolation**: Each test should be independent
2. **Table-driven tests**: Use for multiple similar test cases
3. **Mock external dependencies**: Filesystem, databases
4. **Real fixtures**: Sample SQLite and CacheStorage files
5. **Cross-platform validation**: Test on macOS and Linux

## 2. Unit Testing

### Path Detection Tests

```go
// internal/detect_test.go
package internal

import (
    "os"
    "path/filepath"
    "runtime"
    "testing"
)

func TestDetectStoragePaths(t *testing.T) {
    tests := []struct {
        name     string
        goos     string
        home     string
        want     StoragePaths
        wantErr  bool
    }{
        {
            name: "macOS path",
            goos: "darwin",
            home: "/Users/test",
            want: StoragePaths{
                Workspace: "/Users/test/Library/Application Support/Cursor/User/workspaceStorage",
                Web:       "/Users/test/Library/Application Support/Cursor/User/WebStorage",
            },
            wantErr: false,
        },
        {
            name: "Linux path",
            goos: "linux",
            home: "/home/test",
            want: StoragePaths{
                Workspace: "/home/test/.config/Cursor/User/workspaceStorage",
                Web:       "/home/test/.config/Cursor/User/WebStorage",
            },
            wantErr: false,
        },
        {
            name: "Unsupported OS",
            goos: "windows",
            home: "/Users/test",
            want: StoragePaths{},
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Mock runtime.GOOS
            oldGOOS := runtime.GOOS
            runtime.GOOS = tt.goos
            defer func() { runtime.GOOS = oldGOOS }()

            // Mock os.UserHomeDir
            // ... implementation

            got, err := DetectStoragePaths()
            if (err != nil) != tt.wantErr {
                t.Errorf("DetectStoragePaths() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("DetectStoragePaths() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### JSON Parsing Tests

```go
// internal/sqlite_reader_test.go
package internal

import (
    "testing"
    "github.com/tidwall/gjson"
)

func TestExtractMessages(t *testing.T) {
    tests := []struct {
        name    string
        json    string
        want    []Message
        wantErr bool
    }{
        {
            name: "valid messages",
            json: `{"messages":[{"role":"user","content":"Hello"},{"role":"assistant","content":"Hi"}]}`,
            want: []Message{
                {Actor: "user", Content: "Hello"},
                {Actor: "assistant", Content: "Hi"},
            },
            wantErr: false,
        },
        {
            name: "missing messages key",
            json: `{"data":[]}`,
            want: nil,
            wantErr: true,
        },
        {
            name: "empty messages",
            json: `{"messages":[]}`,
            want: nil,
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ExtractMessages(tt.json)
            if (err != nil) != tt.wantErr {
                t.Errorf("ExtractMessages() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if !compareMessages(got, tt.want) {
                t.Errorf("ExtractMessages() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Normalization Tests

```go
// internal/model_test.go
package internal

import "testing"

func TestNormalizeActor(t *testing.T) {
    tests := []struct {
        input string
        want  string
    }{
        {"user", "user"},
        {"USER", "user"},
        {"assistant", "assistant"},
        {"ai", "assistant"},
        {"bot", "assistant"},
        {"tool", "tool"},
        {"function", "tool"},
        {"unknown", "user"}, // default
    }

    for _, tt := range tests {
        t.Run(tt.input, func(t *testing.T) {
            if got := normalizeActor(tt.input); got != tt.want {
                t.Errorf("normalizeActor() = %v, want %v", got, tt.want)
            }
        })
    }
}

func TestNormalizeTimestamp(t *testing.T) {
    tests := []struct {
        name  string
        input string
        want  string
    }{
        {"RFC3339", "2025-11-07T14:23:01Z", "2025-11-07T14:23:01Z"},
        {"empty", "", ""},
        {"invalid", "not-a-date", "not-a-date"}, // preserve invalid
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            if got := normalizeTimestamp(tt.input); got != tt.want {
                t.Errorf("normalizeTimestamp() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

## 3. Integration Testing

### Mock Filesystem

```go
// testutil/mockfs.go
package testutil

import (
    "os"
    "path/filepath"
    "testing"
)

func CreateMockCursorDir(t *testing.T) string {
    tmpDir := t.TempDir()

    // Create workspaceStorage structure
    workspaceDir := filepath.Join(tmpDir, "workspaceStorage", "workspace-hash")
    os.MkdirAll(workspaceDir, 0755)

    // Create SQLite database (simplified)
    dbPath := filepath.Join(workspaceDir, "state.vscdb")
    createMockSQLiteDB(t, dbPath)

    // Create WebStorage structure
    webDir := filepath.Join(tmpDir, "WebStorage", "1", "CacheStorage", "cache-name")
    os.MkdirAll(webDir, 0755)

    // Create mock cache file
    cachePath := filepath.Join(webDir, "data_0")
    createMockCacheFile(t, cachePath)

    return tmpDir
}

func createMockSQLiteDB(t *testing.T, path string) {
    // Create minimal SQLite database with test data
    // Use modernc.org/sqlite to create test database
}

func createMockCacheFile(t *testing.T, path string) {
    // Create binary file with embedded JSON
    jsonData := `{"messages":[{"role":"user","content":"test"}]}`
    // Embed in binary format
    // Write to file
}
```

### Integration Test Example

```go
// internal/integration_test.go
package internal

import (
    "testing"
    "github.com/k/cursor-session/testutil"
)

func TestExtractSessionsIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    // Create mock Cursor directory
    mockDir := testutil.CreateMockCursorDir(t)
    defer os.RemoveAll(mockDir)

    paths := StoragePaths{
        Workspace: filepath.Join(mockDir, "workspaceStorage"),
        Web:       filepath.Join(mockDir, "WebStorage"),
    }

    // Extract sessions
    sessions, err := ExtractAllSessions(paths)
    if err != nil {
        t.Fatalf("ExtractAllSessions() error = %v", err)
    }

    if len(sessions) == 0 {
        t.Error("Expected at least one session")
    }

    // Validate session structure
    for _, session := range sessions {
        if session.ID == "" {
            t.Error("Session ID should not be empty")
        }
        if len(session.Messages) == 0 {
            t.Error("Session should have at least one message")
        }
    }
}
```

## 4. Test Fixtures

### Fixture Structure

```
testdata/
├── sqlite/
│   ├── state.vscdb          # Sample SQLite database
│   └── corrupt.vscdb        # Corrupt database for error testing
├── cache/
│   ├── valid_0              # Valid cache file with JSON
│   └── invalid_0             # Invalid cache file
└── expected/
    ├── session.jsonl         # Expected JSONL output
    ├── session.md            # Expected Markdown output
    └── session.html          # Expected HTML output
```

### Creating Fixtures

```go
// scripts/create_fixtures.go
package main

import (
    "database/sql"
    _ "modernc.org/sqlite"
    "os"
    "path/filepath"
)

func createSQLiteFixture() {
    dbPath := "testdata/sqlite/state.vscdb"
    os.MkdirAll(filepath.Dir(dbPath), 0755)

    db, _ := sql.Open("sqlite", dbPath)
    defer db.Close()

    // Create ItemTable
    db.Exec(`CREATE TABLE IF NOT EXISTS ItemTable (key TEXT, value TEXT)`)

    // Insert test data
    testData := `{"messages":[{"role":"user","content":"Hello"},{"role":"assistant","content":"Hi"}]}`
    db.Exec(`INSERT INTO ItemTable (key, value) VALUES (?, ?)`,
        "workbench.panel.aichat.view.aichat.chatdata", testData)
}
```

## 5. Table-Driven Tests

### Export Format Tests

```go
// internal/export_test.go
package internal

import (
    "bytes"
    "testing"
)

func TestExportJSONL(t *testing.T) {
    session := Session{
        ID: "test-session",
        Messages: []Message{
            {Actor: "user", Content: "Hello"},
            {Actor: "assistant", Content: "Hi"},
        },
    }

    var buf bytes.Buffer
    exporter := &JSONLExporter{}

    if err := exporter.Export(session, &buf); err != nil {
        t.Fatalf("Export() error = %v", err)
    }

    output := buf.String()
    if !contains(output, `"actor":"user"`) {
        t.Error("Output should contain user message")
    }
    if !contains(output, `"actor":"assistant"`) {
        t.Error("Output should contain assistant message")
    }
}
```

## 6. Error Handling Tests

```go
// internal/error_test.go
package internal

import (
    "errors"
    "testing"
)

func TestStorageError(t *testing.T) {
    err := &StorageError{
        Path: "/test/path",
        Op:   "open",
        Err:  errors.New("permission denied"),
    }

    if err.Error() == "" {
        t.Error("Error message should not be empty")
    }

    if !errors.Is(err, err.Err) {
        t.Error("Should unwrap to underlying error")
    }
}

func TestGracefulDegradation(t *testing.T) {
    // Test that one corrupt file doesn't stop entire scan
    // ...
}
```

## 7. Benchmark Tests

```go
// internal/benchmark_test.go
package internal

import "testing"

func BenchmarkSQLiteQuery(b *testing.B) {
    db := setupTestDB(b)
    defer db.Close()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = QuerySessions(db)
    }
}

func BenchmarkJSONParsing(b *testing.B) {
    jsonStr := `{"messages":[{"role":"user","content":"test"}]}`

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = ExtractMessages(jsonStr)
    }
}

func BenchmarkExportJSONL(b *testing.B) {
    session := createTestSession(100) // 100 messages

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        var buf bytes.Buffer
        _ = ExportJSONL(session, &buf)
    }
}
```

## 8. Cross-Platform Testing

### CI Configuration

```yaml
# .github/workflows/test.yml
name: Test

on: [push, pull_request]

jobs:
  test:
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest]
        go: [1.22, 1.23]
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: ${{ matrix.go }}
      - run: go test ./...
      - run: go test -race ./...
      - run: go test -bench=. ./...
```

## 9. Test Coverage Goals

- **Unit tests**: > 80% coverage
- **Integration tests**: Cover main workflows
- **Error paths**: Test all error scenarios
- **Edge cases**: Empty data, corrupt files, missing paths

## 10. Test Utilities

### Helper Functions

```go
// testutil/helpers.go
package testutil

func CompareSessions(a, b Session) bool {
    if a.ID != b.ID {
        return false
    }
    if len(a.Messages) != len(b.Messages) {
        return false
    }
    for i := range a.Messages {
        if a.Messages[i] != b.Messages[i] {
            return false
        }
    }
    return true
}

func LoadFixture(path string) ([]byte, error) {
    return os.ReadFile(filepath.Join("testdata", path))
}
```

## 11. Mock Patterns

### Mock Database

```go
// testutil/mockdb.go
type MockDB struct {
    sessions []Session
}

func (m *MockDB) Query(query string, args ...interface{}) (*sql.Rows, error) {
    // Return mock rows
}
```

### Mock Filesystem

```go
// Use afero or similar for filesystem mocking
import "github.com/spf13/afero"

func TestWithMockFS(t *testing.T) {
    fs := afero.NewMemMapFs()
    // Create test structure in memory
}
```

## 12. Next Steps

1. Create test fixtures (SQLite, CacheStorage)
2. Implement unit tests for all modules
3. Write integration tests
4. Set up CI/CD testing
5. Achieve > 80% code coverage
6. Add benchmark tests
7. Test on multiple platforms
