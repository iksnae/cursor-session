# Cursor Storage Format Research

## Overview

This document details the research findings on Cursor Editor's storage formats, including SQLite databases and Chromium CacheStorage structures.

## 1. SQLite Storage

Cursor uses two storage formats: **Legacy workspaceStorage** and **Modern globalStorage**.

### Path Locations

| Storage Type | OS    | Base Path                                    | Full Path                                       |
| ------------ | ----- | -------------------------------------------- | ----------------------------------------------- |
| **Legacy**   | macOS | `~/Library/Application Support/Cursor/User/` | `workspaceStorage/{workspace-hash}/state.vscdb` |
| **Legacy**   | Linux | `~/.config/Cursor/User/`                     | `workspaceStorage/{workspace-hash}/state.vscdb` |
| **Modern**   | macOS | `~/Library/Application Support/Cursor/User/` | `globalStorage/state.vscdb`                     |
| **Modern**   | Linux | `~/.config/Cursor/User/`                     | `globalStorage/state.vscdb`                     |

### Legacy Storage (workspaceStorage)

**Table: ItemTable**

The `ItemTable` stores key-value pairs where:

- `key`: String identifier for the stored item
- `value`: JSON-encoded string containing the actual data

**Key Patterns Identified**:

1. **`workbench.panel.aichat.view.aichat.chatdata`**

   - Main chat panel data (legacy format)
   - Structure: Contains `tabs[]` array with chat sessions
   - Each tab has: `id`, `title`, `timestamp`, `bubbles[]`

2. **`composer.composerData`**
   - Composer panel chat data (note: `composer.` prefix, not just `composerData`)
   - Structure: Contains `allComposers[]` array
   - Each composer has: `composerId`, `conversation[]`, `text`, `name`, `lastUpdatedAt`, `createdAt`

### Modern Storage (globalStorage)

**Table: cursorDiskKV**

The `cursorDiskKV` table stores key-value pairs for global chat data:

- `key`: String identifier with colon-separated segments
- `value`: JSON-encoded string containing the actual data

**Key Patterns**:

1. **`bubbleId:<chatId>:<bubbleId>`**

   - Individual message bubbles
   - Structure: `{ text, richText, codeBlocks[], timestamp, type }`
   - `type`: 1 = user, 2 = assistant
   - `text`: Primary text content
   - `richText`: JSON structure with nested text nodes (fallback)
   - `codeBlocks[]`: Array of code block objects

2. **`composerData:<composerId>`**

   - Composer conversations
   - Structure: `{ composerId, name, fullConversationHeadersOnly[], conversation[], lastUpdatedAt, createdAt }`
   - `fullConversationHeadersOnly[]`: Array of message headers with `bubbleId` and `type`

3. **`messageRequestContext:<composerId>:<contextId>`**

   - Context data for messages
   - Structure: `{ bubbleId, gitStatusRaw, terminalFiles[], attachedFoldersListDirResults[], cursorRules[], projectLayouts[] }`

4. **`codeBlockDiff:<chatId>:<diffId>`**
   - Code changes and tool actions
   - Structure: `{ newModelDiffWrtV0[], originalModelDiffWrtV0[], filePath, command, toolName, parameters, result }`

### Legacy JSON Structure (ItemTable)

**Chat Data** (`workbench.panel.aichat.view.aichat.chatdata`):

```json
{
  "tabs": [
    {
      "id": "chat-id",
      "title": "Chat Title",
      "timestamp": 1699374181000,
      "bubbles": [
        {
          "type": "user",
          "text": "What is Cursor?",
          "timestamp": 1699374181000
        },
        {
          "type": "ai",
          "text": "Cursor is an AI coding editor...",
          "timestamp": 1699374183000
        }
      ]
    }
  ]
}
```

**Composer Data** (`composer.composerData`):

```json
{
  "allComposers": [
    {
      "composerId": "composer-id",
      "name": "Conversation Name",
      "text": "User message text",
      "conversation": [
        {
          "type": 1,
          "bubbleId": "bubble-id",
          "text": "User message",
          "timestamp": 1699374181000
        }
      ],
      "lastUpdatedAt": 1699374181000,
      "createdAt": 1699374181000
    }
  ]
}
```

### Modern JSON Structure (cursorDiskKV)

**Bubble** (`bubbleId:<chatId>:<bubbleId>`):

```json
{
  "text": "Message text content",
  "richText": "{\"root\":{\"children\":[...]}}",
  "codeBlocks": [
    {
      "language": "go",
      "content": "package main\n\nfunc main() {}"
    }
  ],
  "timestamp": 1699374181000,
  "type": 1
}
```

**Composer Data** (`composerData:<composerId>`):

```json
{
  "composerId": "composer-id",
  "name": "Conversation Name",
  "fullConversationHeadersOnly": [
    {
      "bubbleId": "bubble-id-1",
      "type": 1
    },
    {
      "bubbleId": "bubble-id-2",
      "type": 2
    }
  ],
  "lastUpdatedAt": 1699374181000,
  "createdAt": 1699374181000
}
```

### Query Strategy

**Legacy Storage (ItemTable)**:

```sql
SELECT key, value
FROM ItemTable
WHERE key IN (
  'workbench.panel.aichat.view.aichat.chatdata',
  'composer.composerData'
);
```

**Modern Storage (cursorDiskKV)**:

```sql
-- Get all bubbles for a chat
SELECT key, value
FROM cursorDiskKV
WHERE key LIKE 'bubbleId:%';

-- Get composer data
SELECT key, value
FROM cursorDiskKV
WHERE key LIKE 'composerData:%';

-- Get message context
SELECT key, value
FROM cursorDiskKV
WHERE key LIKE 'messageRequestContext:%';

-- Get code block diffs
SELECT key, value
FROM cursorDiskKV
WHERE key LIKE 'codeBlockDiff:%';
```

### Session ID Generation

- Session IDs may be embedded in the JSON metadata
- If missing, generate UUID v4 or hash-based ID from first message timestamp + content hash
- Format: `session_{timestamp}_{hash}` or UUID

### Workspace Association

- Each `state.vscdb` file is in a workspace-specific subdirectory
- Workspace hash can be extracted from path: `workspaceStorage/{hash}/state.vscdb`
- Store workspace path/hash in Session struct for traceability

## 2. CacheStorage (WebStorage)

### Path Locations

| OS    | Base Path                                               | Cache Path Pattern                      |
| ----- | ------------------------------------------------------- | --------------------------------------- |
| macOS | `~/Library/Application Support/Cursor/User/WebStorage/` | `{origin}/CacheStorage/{cache-name}/_0` |
| Linux | `~/.config/Cursor/User/WebStorage/`                     | `{origin}/CacheStorage/{cache-name}/_0` |

### File Structure

CacheStorage uses Chromium's cache format:

- **Index files**: `index`, `index-dir`, `the-real-index`
- **Data files**: `*_0` binary files containing cached data

### Binary Format

The `_0` files are binary blobs that may contain:

- HTTP cache entries
- IndexedDB data
- JSON fragments embedded in binary format

### Parsing Strategy (Heuristic)

1. Read file as binary
2. Search for JSON-like patterns:
   - Look for `{` and `}` characters
   - Extract substring between first `{` and last `}`
   - Attempt JSON unmarshal
3. Validate extracted JSON:
   - Check for `messages` key
   - Verify `role` and `content` fields exist
   - Extract message arrays

### Limitations

- Heuristic parsing may miss valid data
- Some files are pure binary blobs without JSON
- No structured IndexedDB parser (future enhancement)
- May require full Chromium IndexedDB API for complete extraction

### Alternative Parsing Methods

**Future Enhancement: IndexedDB Parser**

Research needed on:

- Chromium IndexedDB format specification
- Go libraries for IndexedDB parsing
- LevelDB format (underlying storage for IndexedDB)

## 3. Storage Path Detection

### Implementation Strategy

```go
func DetectStoragePaths() (StoragePaths, error) {
    var basePath string
    switch runtime.GOOS {
    case "darwin":
        basePath = filepath.Join(os.Getenv("HOME"),
            "Library/Application Support/Cursor/User")
    case "linux":
        basePath = filepath.Join(os.Getenv("HOME"),
            ".config/Cursor/User")
    default:
        return StoragePaths{}, fmt.Errorf("unsupported OS: %s", runtime.GOOS)
    }

    return StoragePaths{
        Workspace: filepath.Join(basePath, "workspaceStorage"),
        Web:       filepath.Join(basePath, "WebStorage"),
    }, nil
}
```

### Path Variations

- Workspace-specific subdirectories in `workspaceStorage/`
- Multiple origins in `WebStorage/` (e.g., `1/`, `file__0/`)
- CacheStorage subdirectories vary by Cursor version

## 4. Data Extraction Flow

### Legacy Format

1. **Detect paths** → Resolve OS-specific base paths
2. **Scan workspaceStorage** → Walk directories, find `state.vscdb` files
3. **Query ItemTable** → Extract keys: `workbench.panel.aichat.view.aichat.chatdata`, `composer.composerData`
4. **Parse JSON** → Extract tabs/bubbles or allComposers/conversation arrays
5. **Normalize** → Convert to unified Session format

### Modern Format

1. **Detect paths** → Resolve OS-specific base paths
2. **Check globalStorage** → Find `globalStorage/state.vscdb`
3. **Query cursorDiskKV** → Extract:
   - `bubbleId:*` → Build bubble map
   - `composerData:*` → Get conversation headers
   - `messageRequestContext:*` → Get context data
   - `codeBlockDiff:*` → Get tool actions
4. **Reconstruct conversations** → Match bubbles to composers using `fullConversationHeadersOnly`
5. **Extract text** → From `bubble.text` or parse `bubble.richText` JSON
6. **Normalize** → Convert to unified Session format

### Text Extraction Strategy (from cursor-chat-browser)

````go
func extractTextFromBubble(bubble map[string]interface{}) string {
    // 1. Try primary text field
    if text, ok := bubble["text"].(string); ok && text != "" {
        return text
    }

    // 2. Parse richText JSON structure
    if richText, ok := bubble["richText"].(string); ok {
        var richTextData map[string]interface{}
        if json.Unmarshal([]byte(richText), &richTextData) == nil {
            if root, ok := richTextData["root"].(map[string]interface{}); ok {
                if children, ok := root["children"].([]interface{}); ok {
                    return extractTextFromRichText(children)
                }
            }
        }
    }

    // 3. Append code blocks
    if codeBlocks, ok := bubble["codeBlocks"].([]interface{}); ok {
        text := ""
        for _, block := range codeBlocks {
            if blockMap, ok := block.(map[string]interface{}); ok {
                lang := blockMap["language"].(string)
                content := blockMap["content"].(string)
                text += fmt.Sprintf("\n\n```%s\n%s\n```", lang, content)
            }
        }
        return text
    }

    return ""
}
````

## 5. Research Questions

### Answered

- ✅ Storage locations confirmed (macOS/Linux)
- ✅ SQLite table structure (ItemTable)
- ✅ Key patterns identified
- ✅ Basic JSON structure understood

### Pending Investigation

- ⚠️ Exact JSON schema variations across Cursor versions
- ⚠️ Session metadata fields (timestamps, IDs)
- ⚠️ CacheStorage IndexedDB structure details
- ⚠️ Workspace hash generation algorithm
- ⚠️ GlobalStorage format (mentioned in POC limitations)

## 6. Example Data Structures

### SQLite Value Example

```json
{
  "messages": [
    {
      "role": "user",
      "content": "Explain Go routines",
      "timestamp": "2025-11-07T14:23:01Z"
    },
    {
      "role": "assistant",
      "content": "Go routines are lightweight threads...",
      "timestamp": "2025-11-07T14:23:03Z"
    }
  ],
  "sessionId": "abc123",
  "workspace": "/path/to/project"
}
```

### CacheStorage JSON Fragment

```json
{
  "messages": [
    { "role": "user", "content": "..." },
    { "role": "assistant", "content": "..." }
  ]
}
```

## 7. Parsing Strategies

### SQLite Parser

- Use `modernc.org/sqlite` for pure-Go access
- Query ItemTable with key filters
- Parse JSON using `gjson` for path-based extraction
- Handle missing/null values gracefully

### CacheStorage Parser

- Read binary files in chunks
- Search for JSON delimiters (`{`, `}`)
- Extract and validate JSON fragments
- Skip files without valid JSON
- Log warnings for unparseable files

## 8. Next Steps

1. Create test fixtures with sample SQLite databases
2. Create sample CacheStorage binary files
3. Implement parser prototypes
4. Validate against real Cursor installations
5. Document edge cases and error scenarios
