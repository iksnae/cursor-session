# Performance & Optimization Research

## Overview

This document outlines performance considerations, concurrency patterns, and optimization strategies for the Cursor Session Export CLI.

## 1. Performance Targets

### Estimated Performance

| Operation                 | Target Time         | Notes                          |
| ------------------------- | ------------------- | ------------------------------ |
| SQLite scan               | < 100ms per DB      | Single query, small result set |
| JSON parsing              | < 10ms per session  | gjson path extraction          |
| CacheStorage scan         | 50-200ms per file   | Binary read + heuristic parse  |
| Export (JSONL)            | < 50ms per session  | Stream writing                 |
| Export (Markdown)         | < 100ms per session | Template rendering             |
| Full scan (10 workspaces) | < 5 seconds         | Parallel processing            |

## 2. Concurrency Patterns

### Bounded Concurrency

Use worker pool pattern for parallel scanning:

```go
func ScanWorkspacesParallel(workspaces []string) []Session {
    numWorkers := runtime.NumCPU()
    if numWorkers > len(workspaces) {
        numWorkers = len(workspaces)
    }

    jobs := make(chan string, len(workspaces))
    results := make(chan []Session, len(workspaces))

    // Start workers
    var wg sync.WaitGroup
    for i := 0; i < numWorkers; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for workspace := range jobs {
                sessions, _ := ScanWorkspace(workspace)
                results <- sessions
            }
        }()
    }

    // Send jobs
    for _, workspace := range workspaces {
        jobs <- workspace
    }
    close(jobs)

    // Wait for completion
    go func() {
        wg.Wait()
        close(results)
    }()

    // Collect results
    var allSessions []Session
    for sessions := range results {
        allSessions = append(allSessions, sessions...)
    }

    return allSessions
}
```

### Channel-Based Coordination

```go
type ScanResult struct {
    Sessions []Session
    Err      error
}

func ScanWithResults(workspaces []string) ([]Session, []error) {
    results := make(chan ScanResult, len(workspaces))

    for _, workspace := range workspaces {
        go func(w string) {
            sessions, err := ScanWorkspace(w)
            results <- ScanResult{Sessions: sessions, Err: err}
        }(workspace)
    }

    var allSessions []Session
    var errors []error

    for i := 0; i < len(workspaces); i++ {
        result := <-results
        if result.Err != nil {
            errors = append(errors, result.Err)
        } else {
            allSessions = append(allSessions, result.Sessions...)
        }
    }

    return allSessions, errors
}
```

## 3. SQLite Performance

### Connection Reuse

```go
type DBScanner struct {
    db *sql.DB
}

func NewDBScanner() *DBScanner {
    return &DBScanner{}
}

func (s *DBScanner) ScanWorkspace(path string) ([]Session, error) {
    dbPath := filepath.Join(path, "state.vscdb")

    // Reuse connection if same database
    if s.db == nil || s.lastPath != dbPath {
        if s.db != nil {
            s.db.Close()
        }
        db, err := sql.Open("sqlite", dbPath)
        if err != nil {
            return nil, err
        }
        s.db = db
        s.lastPath = dbPath
    }

    return QuerySessions(s.db)
}
```

### Query Optimization

```go
// Use prepared statements for repeated queries
func QuerySessionsOptimized(db *sql.DB) ([]Session, error) {
    stmt, err := db.Prepare(`
        SELECT key, value
        FROM ItemTable
        WHERE key IN (?, ?, ?)
    `)
    if err != nil {
        return nil, err
    }
    defer stmt.Close()

    rows, err := stmt.Query(
        "aiService.prompts",
        "workbench.panel.aichat.view.aichat.chatdata",
        "composerData")
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    // Process rows...
}
```

### Index Considerations

- SQLite automatically indexes primary keys
- ItemTable likely has key as primary key
- No additional indexes needed for this use case

## 4. JSON Parsing Performance

### gjson vs encoding/json

**Benchmark Results** (estimated):

```
BenchmarkGJSON-8         1000000    1200 ns/op    512 B/op    2 allocs/op
BenchmarkStdJSON-8        500000    2400 ns/op   2048 B/op    5 allocs/op
```

**Recommendation**: Use `gjson` for extraction, `encoding/json` for serialization.

### Streaming JSON Parsing

For very large JSON values:

```go
import "github.com/tidwall/gjson"

func ExtractMessagesStreaming(jsonStr string) []Message {
    var messages []Message

    // Use ForEach for memory efficiency
    gjson.Get(jsonStr, "messages").ForEach(func(_, msg gjson.Result) bool {
        messages = append(messages, Message{
            Actor:   msg.Get("role").String(),
            Content: msg.Get("content").String(),
        })
        return true
    })

    return messages
}
```

## 5. File I/O Optimization

### Buffered Reading

```go
func ReadCacheFileBuffered(path string) ([]byte, error) {
    file, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    // Use buffered reader for large files
    reader := bufio.NewReader(file)
    return io.ReadAll(reader)
}
```

### Streaming for Large Files

```go
func ParseCacheFileStreaming(path string) (Session, error) {
    file, err := os.Open(path)
    if err != nil {
        return Session{}, err
    }
    defer file.Close()

    // Read in chunks to find JSON
    buffer := make([]byte, 4096)
    var jsonStart, jsonEnd int

    for {
        n, err := file.Read(buffer)
        if err == io.EOF {
            break
        }
        // Search for JSON delimiters...
    }

    // Extract and parse JSON...
}
```

## 6. Export Performance

### Streaming Export

```go
func ExportJSONLStreaming(sessions []Session, w io.Writer) error {
    enc := json.NewEncoder(w)

    for _, session := range sessions {
        for _, msg := range session.Messages {
            if err := enc.Encode(msg); err != nil {
                return err
            }
        }
    }

    return nil
}
```

### Concurrent Export

```go
func ExportSessionsParallel(sessions []Session, format string, outputDir string) error {
    numWorkers := runtime.NumCPU()
    jobs := make(chan Session, len(sessions))
    errors := make(chan error, len(sessions))

    var wg sync.WaitGroup
    for i := 0; i < numWorkers; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            exporter, _ := NewExporter(format)
            for session := range jobs {
                filename := fmt.Sprintf("session_%s.%s", session.ID, getExtension(format))
                filepath := filepath.Join(outputDir, filename)

                file, err := os.Create(filepath)
                if err != nil {
                    errors <- err
                    continue
                }

                if err := exporter.Export(session, file); err != nil {
                    errors <- err
                }
                file.Close()
            }
        }()
    }

    // Send jobs
    for _, session := range sessions {
        jobs <- session
    }
    close(jobs)

    // Wait and collect errors
    go func() {
        wg.Wait()
        close(errors)
    }()

    var errs []error
    for err := range errors {
        errs = append(errs, err)
    }

    if len(errs) > 0 {
        return fmt.Errorf("export errors: %v", errs)
    }

    return nil
}
```

## 7. Memory Optimization

### Avoid Loading All Data

```go
// Process sessions one at a time instead of loading all
func ProcessSessionsStreaming(paths StoragePaths, exporter Exporter, outputDir string) error {
    // SQLite sessions
    sqliteSessions := make(chan Session)
    go func() {
        defer close(sqliteSessions)
        // Extract and send sessions one by one
    }()

    // CacheStorage sessions
    cacheSessions := make(chan Session)
    go func() {
        defer close(cacheSessions)
        // Extract and send sessions one by one
    }()

    // Merge and export
    for {
        select {
        case session, ok := <-sqliteSessions:
            if !ok {
                sqliteSessions = nil
            } else {
                exportSession(session, exporter, outputDir)
            }
        case session, ok := <-cacheSessions:
            if !ok {
                cacheSessions = nil
            } else {
                exportSession(session, exporter, outputDir)
            }
        }

        if sqliteSessions == nil && cacheSessions == nil {
            break
        }
    }

    return nil
}
```

### Object Pooling

For high-frequency operations:

```go
var messagePool = sync.Pool{
    New: func() interface{} {
        return &Message{}
    },
}

func GetMessage() *Message {
    return messagePool.Get().(*Message)
}

func PutMessage(m *Message) {
    m.Timestamp = ""
    m.Actor = ""
    m.Content = ""
    messagePool.Put(m)
}
```

## 8. Caching Strategies

### Session Deduplication Cache

```go
type SessionCache struct {
    seen map[string]bool
    mu   sync.Mutex
}

func (c *SessionCache) IsSeen(session Session) bool {
    hash := hashSessionContent(session)

    c.mu.Lock()
    defer c.mu.Unlock()

    if c.seen[hash] {
        return true
    }

    c.seen[hash] = true
    return false
}
```

### Path Cache

Cache detected paths to avoid repeated filesystem operations:

```go
var pathCache = struct {
    paths StoragePaths
    once  sync.Once
}{}

func GetStoragePaths() StoragePaths {
    pathCache.once.Do(func() {
        pathCache.paths, _ = DetectStoragePaths()
    })
    return pathCache.paths
}
```

## 9. Benchmarking

### Benchmark Suite

```go
// internal/benchmark_test.go
package internal

import "testing"

func BenchmarkScanWorkspace(b *testing.B) {
    workspace := setupTestWorkspace(b)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = ScanWorkspace(workspace)
    }
}

func BenchmarkParseCacheFile(b *testing.B) {
    cacheFile := setupTestCacheFile(b)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = ParseCacheFile(cacheFile)
    }
}

func BenchmarkExportJSONL(b *testing.B) {
    session := createTestSession(100)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        var buf bytes.Buffer
        _ = ExportJSONL(session, &buf)
    }
}

func BenchmarkConcurrentScan(b *testing.B) {
    workspaces := setupTestWorkspaces(b, 10)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = ScanWorkspacesParallel(workspaces)
    }
}
```

## 10. Profiling

### CPU Profiling

```go
import _ "net/http/pprof"
import "net/http"

func main() {
    go func() {
        log.Println(http.ListenAndServe("localhost:6060", nil))
    }()
    // ... rest of program
}
```

Run with:

```bash
go test -cpuprofile=cpu.prof -bench=.
go tool pprof cpu.prof
```

### Memory Profiling

```bash
go test -memprofile=mem.prof -bench=.
go tool pprof mem.prof
```

## 11. Optimization Checklist

- [ ] Use bounded concurrency (runtime.NumCPU())
- [ ] Reuse database connections when possible
- [ ] Use gjson for JSON extraction
- [ ] Stream large file operations
- [ ] Export sessions concurrently
- [ ] Cache detected paths
- [ ] Avoid loading all sessions into memory
- [ ] Use prepared statements for SQLite
- [ ] Profile and optimize hot paths

## 12. Performance Monitoring

### Metrics to Track

- Number of databases scanned
- Number of cache files processed
- Total sessions extracted
- Export time per format
- Memory usage
- Error rates

### Logging Performance

```go
func ScanWithMetrics(paths StoragePaths) ([]Session, Metrics) {
    start := time.Now()
    var metrics Metrics

    sessions, err := ScanWorkspaceStorage(paths.Workspace)
    metrics.SQLiteScanTime = time.Since(start)
    metrics.SQLiteSessions = len(sessions)

    start = time.Now()
    cacheSessions, err := ScanCacheStorage(paths.Web)
    metrics.CacheScanTime = time.Since(start)
    metrics.CacheSessions = len(cacheSessions)

    metrics.TotalSessions = len(sessions) + len(cacheSessions)
    metrics.TotalTime = time.Since(start)

    return append(sessions, cacheSessions...), metrics
}
```

## 13. Next Steps

1. Implement concurrent scanning
2. Add performance benchmarks
3. Profile hot paths
4. Optimize based on profiling results
5. Document performance characteristics
6. Set up continuous performance monitoring
