# Error Handling & Logging Strategy

## Overview

This document defines the error handling and logging strategy for the Cursor Session Export CLI, ensuring robust operation and graceful degradation.

## 1. Error Types

### Custom Error Types

```go
// StorageError represents errors accessing storage files
type StorageError struct {
    Path string
    Op   string // "open", "read", "parse"
    Err  error
}

func (e *StorageError) Error() string {
    return fmt.Sprintf("storage error: %s %s: %v", e.Op, e.Path, e.Err)
}

func (e *StorageError) Unwrap() error {
    return e.Err
}

// ParseError represents errors parsing data
type ParseError struct {
    Source string // "sqlite", "cache"
    Key    string // storage key or file path
    Err    error
}

func (e *ParseError) Error() string {
    return fmt.Sprintf("parse error [%s] %s: %v", e.Source, e.Key, e.Err)
}

func (e *ParseError) Unwrap() error {
    return e.Err
}

// ExportError represents errors during export
type ExportError struct {
    Format string
    Path   string
    Err    error
}

func (e *ExportError) Error() string {
    return fmt.Sprintf("export error [%s] %s: %v", e.Format, e.Path, e.Err)
}

func (e *ExportError) Unwrap() error {
    return e.Err
}
```

## 2. Error Handling Patterns

### Error Wrapping

Always wrap errors with context:

```go
func OpenDatabase(path string) (*sql.DB, error) {
    db, err := sql.Open("sqlite", path)
    if err != nil {
        return nil, &StorageError{
            Path: path,
            Op:   "open",
            Err:  fmt.Errorf("failed to open database: %w", err),
        }
    }

    if err := db.Ping(); err != nil {
        db.Close()
        return nil, &StorageError{
            Path: path,
            Op:   "ping",
            Err:  fmt.Errorf("database connection failed: %w", err),
        }
    }

    return db, nil
}
```

### Graceful Degradation

Continue processing when possible:

```go
func ScanWorkspaceStorage(root string) []Session {
    var allSessions []Session

    // Find databases
    dbFiles, err := findDatabases(root)
    if err != nil {
        log.Printf("Warning: failed to scan %s: %v", root, err)
        return allSessions // Return empty, don't fail
    }

    // Process each database
    for _, dbPath := range dbFiles {
        sessions, err := extractSessionsFromDB(dbPath)
        if err != nil {
            log.Printf("Warning: failed to parse %s: %v", dbPath, err)
            continue // Skip this database, continue with others
        }
        allSessions = append(allSessions, sessions...)
    }

    return allSessions
}
```

### Partial Failure Handling

```go
func ExtractSessionsFromDB(dbPath string) ([]Session, error) {
    db, err := OpenDatabase(dbPath)
    if err != nil {
        return nil, err // Fail fast for critical errors
    }
    defer db.Close()

    var sessions []Session
    var errors []error

    rows, err := db.Query("SELECT key, value FROM ItemTable WHERE key IN (?, ?, ?)",
        "aiService.prompts",
        "workbench.panel.aichat.view.aichat.chatdata",
        "composerData")
    if err != nil {
        return nil, fmt.Errorf("query failed: %w", err)
    }
    defer rows.Close()

    for rows.Next() {
        var key, value string
        if err := rows.Scan(&key, &value); err != nil {
            errors = append(errors, fmt.Errorf("scan failed for key %s: %w", key, err))
            continue
        }

        session, err := NormalizeSQLiteSession(key, value, dbPath)
        if err != nil {
            errors = append(errors, fmt.Errorf("normalize failed for key %s: %w", key, err))
            continue
        }

        sessions = append(sessions, session)
    }

    // Log errors but return successful sessions
    if len(errors) > 0 {
        log.Printf("Warning: %d errors encountered while processing %s", len(errors), dbPath)
        for _, err := range errors {
            log.Printf("  - %v", err)
        }
    }

    return sessions, rows.Err()
}
```

## 3. Specific Error Scenarios

### Missing Paths

```go
func DetectStoragePaths() (StoragePaths, error) {
    home, err := os.UserHomeDir()
    if err != nil {
        return StoragePaths{}, fmt.Errorf("failed to get home directory: %w", err)
    }

    var basePath string
    switch runtime.GOOS {
    case "darwin":
        basePath = filepath.Join(home, "Library/Application Support/Cursor/User")
    case "linux":
        basePath = filepath.Join(home, ".config/Cursor/User")
    default:
        return StoragePaths{}, fmt.Errorf("unsupported OS: %s", runtime.GOOS)
    }

    // Check if paths exist (warn but don't fail)
    workspacePath := filepath.Join(basePath, "workspaceStorage")
    if _, err := os.Stat(workspacePath); os.IsNotExist(err) {
        log.Printf("Warning: workspaceStorage not found at %s", workspacePath)
    }

    webPath := filepath.Join(basePath, "WebStorage")
    if _, err := os.Stat(webPath); os.IsNotExist(err) {
        log.Printf("Warning: WebStorage not found at %s", webPath)
    }

    return StoragePaths{
        Workspace: workspacePath,
        Web:       webPath,
    }, nil
}
```

### Corrupt Database Files

```go
func OpenDatabase(path string) (*sql.DB, error) {
    // Check file exists and is readable
    info, err := os.Stat(path)
    if err != nil {
        return nil, &StorageError{
            Path: path,
            Op:   "stat",
            Err:  err,
        }
    }

    if info.Size() == 0 {
        return nil, &StorageError{
            Path: path,
            Op:   "open",
            Err:  fmt.Errorf("database file is empty"),
        }
    }

    db, err := sql.Open("sqlite", path)
    if err != nil {
        return nil, &StorageError{
            Path: path,
            Op:   "open",
            Err:  err,
        }
    }

    // Test connection
    if err := db.Ping(); err != nil {
        db.Close()
        return nil, &StorageError{
            Path: path,
            Op:   "ping",
            Err:  fmt.Errorf("database may be corrupt: %w", err),
        }
    }

    return db, nil
}
```

### Permission Errors

```go
func ScanDirectory(root string) ([]string, error) {
    var files []string

    err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
        if err != nil {
            // Check if it's a permission error
            if os.IsPermission(err) {
                log.Printf("Warning: permission denied: %s", path)
                return nil // Continue walking
            }
            return err // Stop on other errors
        }

        // Process file...
        return nil
    })

    return files, err
}
```

### Non-JSON Content

```go
func ParseCacheFile(path string) (Session, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return Session{}, &StorageError{
            Path: path,
            Op:   "read",
            Err:  err,
        }
    }

    // Try to extract JSON
    jsonStr, found := extractJSONFragment(data)
    if !found {
        return Session{}, &ParseError{
            Source: "cache",
            Key:    path,
            Err:    fmt.Errorf("no JSON fragment found"),
        }
    }

    // Try to parse
    session, err := NormalizeCacheSession(jsonStr, path)
    if err != nil {
        return Session{}, &ParseError{
            Source: "cache",
            Key:    path,
            Err:    err,
        }
    }

    return session, nil
}
```

### Output Directory Issues

```go
func EnsureOutputDir(path string) error {
    info, err := os.Stat(path)
    if err == nil {
        if !info.IsDir() {
            return fmt.Errorf("output path exists but is not a directory: %s", path)
        }
        return nil // Directory exists
    }

    if os.IsNotExist(err) {
        // Create directory
        if err := os.MkdirAll(path, 0755); err != nil {
            return fmt.Errorf("failed to create output directory: %w", err)
        }
        return nil
    }

    return fmt.Errorf("failed to check output directory: %w", err)
}
```

## 4. Logging Strategy

### Log Levels

```go
const (
    LogLevelError = iota
    LogLevelWarn
    LogLevelInfo
    LogLevelDebug
)

var logLevel = LogLevelInfo

func SetLogLevel(level int) {
    logLevel = level
}
```

### Logging Functions

```go
func logError(format string, args ...interface{}) {
    if logLevel >= LogLevelError {
        log.Printf("[ERROR] "+format, args...)
    }
}

func logWarn(format string, args ...interface{}) {
    if logLevel >= LogLevelWarn {
        log.Printf("[WARN] "+format, args...)
    }
}

func logInfo(format string, args ...interface{}) {
    if logLevel >= LogLevelInfo {
        log.Printf("[INFO] "+format, args...)
    }
}

func logDebug(format string, args ...interface{}) {
    if logLevel >= LogLevelDebug {
        log.Printf("[DEBUG] "+format, args...)
    }
}
```

### Structured Logging (Optional)

```go
import "github.com/rs/zerolog/log"

func logWithContext() {
    log.Info().
        Str("path", dbPath).
        Int("sessions", count).
        Str("source", "sqlite").
        Msg("Extracted sessions from database")
}
```

### Progress Reporting

```go
func ExportSessions(sessions []Session, format string, outputDir string) error {
    total := len(sessions)
    logInfo("Exporting %d sessions to %s", total, outputDir)

    for i, session := range sessions {
        if err := exportSession(session, format, outputDir); err != nil {
            logError("Failed to export session %s: %v", session.ID, err)
            continue
        }

        if (i+1)%10 == 0 {
            logInfo("Progress: %d/%d sessions exported", i+1, total)
        }
    }

    logInfo("Export complete: %d sessions exported", total)
    return nil
}
```

## 5. Error Recovery

### Retry Logic

```go
func OpenDatabaseWithRetry(path string, maxRetries int) (*sql.DB, error) {
    var lastErr error

    for i := 0; i < maxRetries; i++ {
        db, err := OpenDatabase(path)
        if err == nil {
            return db, nil
        }

        lastErr = err

        // Check if it's a lock error (Cursor may have DB open)
        if strings.Contains(err.Error(), "database is locked") {
            time.Sleep(time.Second * time.Duration(i+1))
            continue
        }

        // Don't retry on other errors
        return nil, err
    }

    return nil, fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}
```

### Fallback Strategies

```go
func ExtractSessions(paths StoragePaths) []Session {
    var sessions []Session

    // Try SQLite first
    sqliteSessions, err := ExtractFromSQLite(paths.Workspace)
    if err != nil {
        logWarn("SQLite extraction failed: %v", err)
    } else {
        sessions = append(sessions, sqliteSessions...)
    }

    // Try CacheStorage as fallback
    cacheSessions, err := ExtractFromCache(paths.Web)
    if err != nil {
        logWarn("CacheStorage extraction failed: %v", err)
    } else {
        sessions = append(sessions, cacheSessions...)
    }

    return sessions
}
```

## 6. Error Reporting to User

### CLI Error Messages

```go
func handleError(err error) {
    var storageErr *StorageError
    var parseErr *ParseError
    var exportErr *ExportError

    switch {
    case errors.As(err, &storageErr):
        fmt.Fprintf(os.Stderr, "Error accessing storage: %v\n", err)
        fmt.Fprintf(os.Stderr, "  Path: %s\n", storageErr.Path)
        fmt.Fprintf(os.Stderr, "  Operation: %s\n", storageErr.Op)

    case errors.As(err, &parseErr):
        fmt.Fprintf(os.Stderr, "Error parsing data: %v\n", err)
        fmt.Fprintf(os.Stderr, "  Source: %s\n", parseErr.Source)
        fmt.Fprintf(os.Stderr, "  Key: %s\n", parseErr.Key)

    case errors.As(err, &exportErr):
        fmt.Fprintf(os.Stderr, "Error exporting: %v\n", err)
        fmt.Fprintf(os.Stderr, "  Format: %s\n", exportErr.Format)
        fmt.Fprintf(os.Stderr, "  Path: %s\n", exportErr.Path)

    default:
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
    }

    os.Exit(1)
}
```

## 7. Best Practices

1. **Wrap errors with context**: Always add context about what operation failed
2. **Use custom error types**: Make error handling more specific
3. **Continue on non-critical errors**: Don't fail entire operation for single file errors
4. **Log warnings for recoverable errors**: Inform user but continue processing
5. **Fail fast on critical errors**: Database connection failures should stop operation
6. **Provide helpful error messages**: Include paths, operations, and suggestions

## 8. Next Steps

1. Implement error types
2. Add error handling to all functions
3. Create error test cases
4. Validate error messages are helpful
5. Test error recovery scenarios
