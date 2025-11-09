# Cursor Chat Browser Reference Findings

## Overview

This document captures key findings from analyzing the `cursor-chat-browser` project, which provides a JavaScript/TypeScript implementation for reading Cursor chat history. These findings inform our Go implementation strategy.

## Project Reference

**Repository**: `cursor-chat-browser` (located at `~/Projects/cursor-chat-browser`)
**Language**: TypeScript/JavaScript (Next.js)
**SQLite Library**: `better-sqlite3`

## Key Discoveries

### 1. Dual Storage Format Support

Cursor uses **two storage formats** depending on version:

#### Legacy Format (workspaceStorage)

- **Path**: `workspaceStorage/{hash}/state.vscdb`
- **Table**: `ItemTable`
- **Keys**:
  - `workbench.panel.aichat.view.aichat.chatdata` - Chat tabs
  - `composer.composerData` - Composer conversations (note: `composer.` prefix)

#### Modern Format (globalStorage)

- **Path**: `globalStorage/state.vscdb` (sibling to workspaceStorage)
- **Table**: `cursorDiskKV` (not `ItemTable`)
- **Key Patterns**:
  - `bubbleId:<chatId>:<bubbleId>` - Individual message bubbles
  - `composerData:<composerId>` - Composer conversation metadata
  - `messageRequestContext:<composerId>:<contextId>` - Message context
  - `codeBlockDiff:<chatId>:<diffId>` - Code changes and tool actions

### 2. Message Structure (Bubbles)

Messages are stored as "bubbles" with the following structure:

```typescript
interface ChatBubble {
  type: "user" | "ai" | 1 | 2; // 1 = user, 2 = assistant
  text?: string; // Primary text content
  richText?: string; // JSON string with nested structure
  codeBlocks?: Array<{
    // Code blocks in message
    language: string;
    content: string;
  }>;
  timestamp: number; // Unix timestamp in milliseconds
}
```

### 3. Text Extraction Strategy

The reference implementation uses a three-tier extraction strategy:

1. **Primary**: Use `bubble.text` if available
2. **Fallback**: Parse `bubble.richText` JSON structure
3. **Append**: Add code blocks from `bubble.codeBlocks[]` array

**Rich Text Parsing**:

- `richText` is a JSON string containing a nested structure
- Root has `children[]` array
- Each child can be:
  - `type: "text"` with `text` property
  - `type: "code"` with nested `children[]`
- Recursively extract text from all children

### 4. Conversation Reconstruction

Modern format requires **reconstruction** from multiple keys:

1. Get `composerData:<composerId>` → Contains `fullConversationHeadersOnly[]`
2. Each header has `bubbleId` and `type`
3. Look up each `bubbleId` in `bubbleId:<chatId>:<bubbleId>` entries
4. Match bubbles to headers to reconstruct conversation order
5. Sort by `timestamp` to ensure chronological order

### 5. Context Data

Messages can have associated context from `messageRequestContext`:

- `gitStatusRaw` - Git status at time of message
- `terminalFiles[]` - Terminal file references
- `attachedFoldersListDirResults[]` - Folder listings
- `cursorRules[]` - Active Cursor rules
- `projectLayouts[]` - Project structure information

### 6. Tool Actions

Tool actions are stored separately in `codeBlockDiff` entries:

- `newModelDiffWrtV0[]` - Code changes
- `filePath` - File being modified
- `command` - Terminal command executed
- `toolName` - Tool identifier
- `parameters` - Tool parameters (JSON)
- `result` - Tool result (JSON)

### 7. Path Detection

The reference implementation handles multiple platforms:

```typescript
function getDefaultWorkspacePath(): string {
  const home = os.homedir();
  const isWSL = os.release().toLowerCase().includes("microsoft");
  const isRemote = Boolean(process.env.SSH_CONNECTION);

  if (isWSL) {
    return `/mnt/c/Users/${username}/AppData/Roaming/Cursor/User/workspaceStorage`;
  }

  switch (process.platform) {
    case "win32":
      return path.join(home, "AppData/Roaming/Cursor/User/workspaceStorage");
    case "darwin":
      return path.join(
        home,
        "Library/Application Support/Cursor/User/workspaceStorage"
      );
    case "linux":
      if (isRemote) {
        return path.join(home, ".cursor-server/data/User/workspaceStorage");
      }
      return path.join(home, ".config/Cursor/User/workspaceStorage");
  }
}
```

**Key Insight**: Also checks for `globalStorage` as sibling to `workspaceStorage`.

### 8. Workspace Association

Modern format requires **project detection** to associate conversations with workspaces:

1. Read `workspace.json` files to map workspace hashes to folder paths
2. Use `messageRequestContext` → `projectLayouts[]` for primary detection
3. Fallback to file path matching from:
   - `composerData.newlyCreatedFiles[]`
   - `composerData.codeBlockData{}`
   - `bubble.relevantFiles[]`
   - `bubble.attachedFileCodeChunksUris[]`
   - `bubble.context.fileSelections[]`

### 9. Key Extraction Patterns

**Extract chat ID from bubble key**:

```typescript
function extractChatIdFromBubbleKey(key: string): string | null {
  // key format: bubbleId:<chatId>:<bubbleId>
  const match = key.match(/^bubbleId:([^:]+):/);
  return match ? match[1] : null;
}
```

**Extract composer ID from key**:

```typescript
const composerId = row.key.split(":")[1]; // composerData:<composerId>
```

### 10. Data Normalization

The reference implementation normalizes:

- **Type conversion**: `1` → `'user'`, `2` → `'ai'`
- **Timestamp conversion**: Unix milliseconds → ISO8601
- **Text extraction**: Handles missing text, richText fallback, code blocks
- **Context merging**: Combines bubble text with context data

## Implementation Implications for Go CLI

### 1. Support Both Formats

Our Go implementation must:

- Check for both `workspaceStorage` and `globalStorage`
- Query appropriate table (`ItemTable` vs `cursorDiskKV`)
- Handle different key patterns

### 2. Bubble Reconstruction

For modern format:

- Build bubble map from all `bubbleId:*` entries
- Match bubbles to conversation headers
- Sort by timestamp

### 3. Text Extraction

Implement three-tier extraction:

1. `bubble.text` (primary)
2. Parse `bubble.richText` JSON (fallback)
3. Append `codeBlocks[]` (enhancement)

### 4. Rich Text Parsing

Need to parse nested JSON structure:

```go
type RichTextNode struct {
    Type     string        `json:"type"`
    Text     string        `json:"text,omitempty"`
    Children []RichTextNode `json:"children,omitempty"`
}

type RichTextRoot struct {
    Root RichTextNode `json:"root"`
}
```

### 5. Context Integration

Consider including context data in exports:

- Git status
- File references
- Tool actions
- Project layout

### 6. Workspace Detection

Implement project association logic:

- Read `workspace.json` files
- Use `projectLayouts` from context
- Fallback to file path matching

## Code Patterns to Adapt

### SQLite Query Pattern

```go
// Legacy
rows := db.Query("SELECT value FROM ItemTable WHERE key = ?", key)

// Modern
rows := db.Query("SELECT key, value FROM cursorDiskKV WHERE key LIKE ?", pattern)
```

### Bubble Map Construction

```go
bubbleMap := make(map[string]Bubble)
rows := db.Query("SELECT key, value FROM cursorDiskKV WHERE key LIKE 'bubbleId:%'")
for rows.Next() {
    var key, value string
    rows.Scan(&key, &value)
    bubbleId := extractBubbleId(key)  // Split by ':'
    bubble := parseBubble(value)
    bubbleMap[bubbleId] = bubble
}
```

### Conversation Reconstruction

```go
// Get composer data
composerData := getComposerData(composerId)

// Reconstruct from headers
for _, header := range composerData.FullConversationHeadersOnly {
    bubble := bubbleMap[header.BubbleId]
    messages = append(messages, normalizeBubble(bubble, header.Type))
}

// Sort by timestamp
sort.Slice(messages, func(i, j int) bool {
    return messages[i].Timestamp < messages[j].Timestamp
})
```

## Updated Research Priorities

Based on these findings:

1. ✅ **Storage format understanding** - Now complete with dual format support
2. ✅ **Message structure** - Bubbles with text/richText/codeBlocks
3. ⚠️ **Rich text parsing** - Need to implement JSON structure parser
4. ⚠️ **Context integration** - Consider including in exports
5. ⚠️ **Workspace association** - Implement project detection logic

## Next Steps

1. Update data models to support bubble structure
2. Implement rich text parser
3. Add globalStorage support to path detection
4. Implement conversation reconstruction logic
5. Consider context data in export formats
