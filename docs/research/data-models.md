# Data Model & Normalization Design

## Overview

This document defines the unified data structures for representing Cursor chat sessions, regardless of their source (SQLite or CacheStorage).

## 1. Core Data Structures

### Message Struct

```go
type Message struct {
    Timestamp string `json:"timestamp,omitempty"` // ISO8601 format
    Actor     string `json:"actor"`              // "user", "assistant", or "tool"
    Content   string `json:"content"`            // Message text (may contain markdown/code)
}
```

**Field Specifications**:

- **Timestamp**: Optional ISO8601 timestamp (e.g., "2025-11-07T14:23:01Z")

  - May be missing from some sources
  - Generate from session metadata if available
  - Use current time as fallback (with flag indicating estimated)

- **Actor**: Required string identifying message sender

  - Values: `"user"`, `"assistant"`, `"tool"`
  - Normalize from source-specific role names:
    - SQLite: `"user"`, `"assistant"` (from `role` field)
    - CacheStorage: May vary, normalize to standard values

- **Content**: Required message text
  - May contain plain text, markdown, or code blocks
  - Preserve formatting for export
  - Handle multi-line content
  - Escape special characters in JSON export

### Session Struct

```go
type Session struct {
    ID        string    `json:"id"`                  // Unique session identifier
    Workspace string    `json:"workspace,omitempty"` // Workspace path or hash
    Source    string    `json:"source"`              // "sqlite" or "cache"
    Messages  []Message `json:"messages"`            // Ordered message array
    Metadata  Metadata  `json:"metadata,omitempty"`  // Additional session info
}

type Metadata struct {
    Key         string    `json:"key,omitempty"`         // Original storage key
    CreatedAt   string    `json:"created_at,omitempty"` // Session creation time
    UpdatedAt   string    `json:"updated_at,omitempty"` // Last message time
    MessageCount int      `json:"message_count"`        // Total messages
    WorkspaceHash string  `json:"workspace_hash,omitempty"` // Workspace identifier
}
```

**Field Specifications**:

- **ID**: Unique session identifier

  - Generation strategy: UUID v4 or content-based hash
  - Format: `session_{timestamp}_{hash}` or UUID
  - Must be stable across exports (same session = same ID)

- **Workspace**: Optional workspace association

  - Path to workspace directory (if available)
  - Or workspace hash from storage path
  - Empty if global/workspace-agnostic session

- **Source**: Storage source identifier

  - Values: `"sqlite"` or `"cache"`
  - Used for debugging and source attribution

- **Messages**: Ordered array of messages

  - Preserve chronological order
  - Empty array if no messages found
  - Minimum 1 message (user prompt) for valid session

- **Metadata**: Optional additional information
  - Original storage key (for SQLite sessions)
  - Timestamps for session lifecycle
  - Message count for quick reference

## 2. Session ID Generation Strategy

### Option 1: UUID v4 (Recommended for New Sessions)

```go
import "github.com/google/uuid"

func GenerateSessionID() string {
    return uuid.New().String()
}
```

**Pros**: Truly unique, no collisions
**Cons**: Not stable across exports (same session gets different ID)

### Option 2: Content-Based Hash (Recommended for Stability)

```go
import (
    "crypto/sha256"
    "encoding/hex"
    "fmt"
)

func GenerateStableSessionID(messages []Message, source string) string {
    // Hash first message content + source + first timestamp
    h := sha256.New()

    if len(messages) > 0 {
        h.Write([]byte(messages[0].Content))
        h.Write([]byte(messages[0].Timestamp))
    }
    h.Write([]byte(source))

    hash := hex.EncodeToString(h.Sum(nil))[:16]
    timestamp := time.Now().Format("20060102")

    return fmt.Sprintf("session_%s_%s", timestamp, hash)
}
```

**Pros**: Stable across exports (same content = same ID)
**Cons**: Potential collisions (very rare with SHA256)

### Option 3: Hybrid Approach

```go
func GenerateSessionID(session Session) string {
    // If session has existing ID from source, use it
    if session.Metadata.Key != "" {
        // Generate hash from key + first message
        return hashFromKey(session.Metadata.Key, session.Messages)
    }

    // Otherwise generate stable hash
    return GenerateStableSessionID(session.Messages, session.Source)
}
```

**Recommendation**: Use Option 2 (content-based hash) for deduplication and stability.

## 3. Normalization Logic

### SQLite → Session Conversion

```go
func NormalizeSQLiteSession(key string, valueJSON string, workspace string) (Session, error) {
    // Parse JSON value
    messagesJSON := gjson.Get(valueJSON, "messages")
    if !messagesJSON.Exists() {
        return Session{}, fmt.Errorf("no messages found in key %s", key)
    }

    var messages []Message
    messagesJSON.ForEach(func(_, msg gjson.Result) bool {
        role := msg.Get("role").String()
        content := msg.Get("content").String()
        timestamp := msg.Get("timestamp").String()

        // Normalize actor
        actor := normalizeActor(role)

        messages = append(messages, Message{
            Timestamp: normalizeTimestamp(timestamp),
            Actor:     actor,
            Content:   content,
        })
        return true
    })

    if len(messages) == 0 {
        return Session{}, fmt.Errorf("empty message array")
    }

    // Generate session ID
    sessionID := GenerateStableSessionID(messages, "sqlite")

    // Extract metadata
    metadata := Metadata{
        Key:          key,
        MessageCount: len(messages),
        WorkspaceHash: extractWorkspaceHash(workspace),
    }

    // Set timestamps
    if len(messages) > 0 {
        metadata.CreatedAt = messages[0].Timestamp
        metadata.UpdatedAt = messages[len(messages)-1].Timestamp
    }

    return Session{
        ID:        sessionID,
        Workspace: workspace,
        Source:    "sqlite",
        Messages:  messages,
        Metadata:  metadata,
    }, nil
}
```

### CacheStorage → Session Conversion

```go
func NormalizeCacheSession(jsonData string, cachePath string) (Session, error) {
    // Parse extracted JSON fragment
    var data map[string]interface{}
    if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
        return Session{}, fmt.Errorf("failed to parse JSON: %w", err)
    }

    // Extract messages
    messagesRaw, ok := data["messages"].([]interface{})
    if !ok {
        return Session{}, fmt.Errorf("messages not found or invalid type")
    }

    var messages []Message
    for _, msgRaw := range messagesRaw {
        msgMap, ok := msgRaw.(map[string]interface{})
        if !ok {
            continue // Skip invalid messages
        }

        role, _ := msgMap["role"].(string)
        content, _ := msgMap["content"].(string)
        timestamp, _ := msgMap["timestamp"].(string)

        messages = append(messages, Message{
            Timestamp: normalizeTimestamp(timestamp),
            Actor:     normalizeActor(role),
            Content:   content,
        })
    }

    if len(messages) == 0 {
        return Session{}, fmt.Errorf("no valid messages found")
    }

    sessionID := GenerateStableSessionID(messages, "cache")

    return Session{
        ID:       sessionID,
        Source:   "cache",
        Messages: messages,
        Metadata: Metadata{
            MessageCount: len(messages),
        },
    }, nil
}
```

### Helper Functions

```go
func normalizeActor(role string) string {
    switch strings.ToLower(role) {
    case "user", "human":
        return "user"
    case "assistant", "ai", "bot":
        return "assistant"
    case "tool", "function":
        return "tool"
    default:
        return "user" // Default fallback
    }
}

func normalizeTimestamp(ts string) string {
    if ts == "" {
        return "" // Allow empty timestamps
    }

    // Try to parse and reformat to ISO8601
    t, err := time.Parse(time.RFC3339, ts)
    if err != nil {
        // Try other formats
        formats := []string{
            time.RFC3339Nano,
            "2006-01-02T15:04:05Z07:00",
            "2006-01-02 15:04:05",
        }

        for _, format := range formats {
            if t, err := time.Parse(format, ts); err == nil {
                return t.Format(time.RFC3339)
            }
        }

        // Return original if can't parse
        return ts
    }

    return t.Format(time.RFC3339)
}
```

## 4. Handling Missing Fields

### Timestamp Handling

- **Missing timestamp**: Leave empty, or generate from message order
- **Invalid timestamp**: Preserve original string, log warning
- **Multiple formats**: Normalize to ISO8601 when possible

### Actor Normalization

- **Unknown role**: Default to "user"
- **Case variations**: Case-insensitive matching
- **Aliases**: Map common aliases (human→user, ai→assistant)

### Content Handling

- **Empty content**: Skip message or include with placeholder
- **Null content**: Treat as empty string
- **Non-string content**: Convert to string representation

## 5. Deduplication Strategy

### Problem

Same session may appear in both SQLite and CacheStorage.

### Solution

```go
func DeduplicateSessions(sessions []Session) []Session {
    seen := make(map[string]bool)
    var unique []Session

    for _, session := range sessions {
        // Use content hash for deduplication
        hash := hashSessionContent(session)

        if !seen[hash] {
            seen[hash] = true
            unique = append(unique, session)
        } else {
            // Prefer SQLite over CacheStorage
            for i, existing := range unique {
                if hashSessionContent(existing) == hash {
                    if session.Source == "sqlite" && existing.Source == "cache" {
                        unique[i] = session // Replace with SQLite version
                    }
                    break
                }
            }
        }
    }

    return unique
}

func hashSessionContent(session Session) string {
    h := sha256.New()
    for _, msg := range session.Messages {
        h.Write([]byte(msg.Actor))
        h.Write([]byte(msg.Content))
        h.Write([]byte(msg.Timestamp))
    }
    return hex.EncodeToString(h.Sum(nil))
}
```

## 6. Example Transformations

### SQLite Input

```json
{
  "key": "workbench.panel.aichat.view.aichat.chatdata",
  "value": "{\"messages\":[{\"role\":\"user\",\"content\":\"Hello\"},{\"role\":\"assistant\",\"content\":\"Hi!\"}]}"
}
```

### Normalized Output

```json
{
  "id": "session_20251107_a1b2c3d4",
  "workspace": "/path/to/project",
  "source": "sqlite",
  "messages": [
    { "timestamp": "", "actor": "user", "content": "Hello" },
    { "timestamp": "", "actor": "assistant", "content": "Hi!" }
  ],
  "metadata": {
    "key": "workbench.panel.aichat.view.aichat.chatdata",
    "message_count": 2
  }
}
```

## 7. Validation Rules

### Session Validation

- Must have at least 1 message
- All messages must have actor and content
- Session ID must be non-empty
- Source must be "sqlite" or "cache"

### Message Validation

- Actor must be one of: "user", "assistant", "tool"
- Content must be non-empty (or skip message)
- Timestamp should be ISO8601 (warn if invalid)

## 8. Next Steps

1. Implement normalization functions
2. Create test cases for edge cases
3. Validate against real Cursor data
4. Performance test with large message arrays
