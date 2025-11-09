package internal

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// AgentStorageReader reads session data from cursor-agent CLI store.db files
type AgentStorageReader struct {
	storeDBPaths []string
}

// NewAgentStorageReader creates a new AgentStorageReader with the given store.db paths
func NewAgentStorageReader(storeDBPaths []string) *AgentStorageReader {
	return &AgentStorageReader{
		storeDBPaths: storeDBPaths,
	}
}

// QueryBlobsTable queries the blobs table from a store.db file
func QueryBlobsTable(db *sql.DB) ([]BlobEntry, error) {
	// Check if blobs table exists
	var tableExists bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT name FROM sqlite_master 
			WHERE type='table' AND name='blobs'
		)
	`).Scan(&tableExists)
	if err != nil {
		return nil, fmt.Errorf("failed to check for blobs table: %w", err)
	}

	if !tableExists {
		return []BlobEntry{}, nil
	}

	// Query all blobs - we'll need to inspect the schema
	// Common patterns: key-value, id-data, etc.
	// Try to get column names first
	rows, err := db.Query("PRAGMA table_info(blobs)")
	if err != nil {
		return nil, fmt.Errorf("failed to get blobs table info: %w", err)
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull int
		var defaultValue sql.NullString
		var pk int

		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err != nil {
			continue
		}
		columns = append(columns, name)
	}

	if len(columns) == 0 {
		return []BlobEntry{}, nil
	}

	// Build query based on common column patterns
	// Try key-value pattern first (most common for session storage)
	var query string
	if containsString(columns, "key") && containsString(columns, "value") {
		query = "SELECT key, value FROM blobs WHERE value IS NOT NULL"
	} else if containsString(columns, "id") && containsString(columns, "data") {
		query = "SELECT id, data FROM blobs WHERE data IS NOT NULL"
	} else if len(columns) >= 2 {
		// Use first two columns
		query = fmt.Sprintf("SELECT %s, %s FROM blobs WHERE %s IS NOT NULL", columns[0], columns[1], columns[1])
	} else {
		return []BlobEntry{}, nil
	}

	rows, err = db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query blobs table: %w", err)
	}
	defer rows.Close()

	var entries []BlobEntry
	rowCount := 0
	for rows.Next() {
		rowCount++
		var entry BlobEntry
		var value sql.NullString
		if err := rows.Scan(&entry.Key, &value); err != nil {
			LogWarn("Failed to scan blob row %d: %v", rowCount, err)
			continue
		}
		if value.Valid {
			entry.Value = value.String
			entries = append(entries, entry)
			// Log first few entries for diagnostics
			if rowCount <= 3 {
				valuePreview := entry.Value
				if len(valuePreview) > 200 {
					valuePreview = valuePreview[:200] + "..."
				}
				LogInfo("Blob entry %d: key='%s', value_preview='%s'", rowCount, entry.Key, valuePreview)
			}
		} else {
			LogWarn("Blob row %d has NULL value: key='%s'", rowCount, entry.Key)
		}
	}

	LogInfo("QueryBlobsTable: queried %d rows, returned %d valid entries", rowCount, len(entries))

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return entries, nil
}

// QueryMetaTable queries the meta table from a store.db file
func QueryMetaTable(db *sql.DB) ([]MetaEntry, error) {
	// Check if meta table exists
	var tableExists bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT name FROM sqlite_master 
			WHERE type='table' AND name='meta'
		)
	`).Scan(&tableExists)
	if err != nil {
		return nil, fmt.Errorf("failed to check for meta table: %w", err)
	}

	if !tableExists {
		return []MetaEntry{}, nil
	}

	// Query meta table - similar flexible approach
	rows, err := db.Query("PRAGMA table_info(meta)")
	if err != nil {
		return nil, fmt.Errorf("failed to get meta table info: %w", err)
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull int
		var defaultValue sql.NullString
		var pk int

		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err != nil {
			continue
		}
		columns = append(columns, name)
	}

	if len(columns) == 0 {
		return []MetaEntry{}, nil
	}

	var query string
	if containsString(columns, "key") && containsString(columns, "value") {
		query = "SELECT key, value FROM meta WHERE value IS NOT NULL"
	} else if containsString(columns, "id") && containsString(columns, "data") {
		query = "SELECT id, data FROM meta WHERE data IS NOT NULL"
	} else if len(columns) >= 2 {
		query = fmt.Sprintf("SELECT %s, %s FROM meta WHERE %s IS NOT NULL", columns[0], columns[1], columns[1])
	} else {
		return []MetaEntry{}, nil
	}

	rows, err = db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query meta table: %w", err)
	}
	defer rows.Close()

	var entries []MetaEntry
	rowCount := 0
	for rows.Next() {
		rowCount++
		var entry MetaEntry
		var value sql.NullString
		if err := rows.Scan(&entry.Key, &value); err != nil {
			LogWarn("Failed to scan meta row %d: %v", rowCount, err)
			continue
		}
		if value.Valid {
			entry.Value = value.String
			entries = append(entries, entry)
			// Log first few entries for diagnostics
			if rowCount <= 3 {
				valuePreview := entry.Value
				if len(valuePreview) > 200 {
					valuePreview = valuePreview[:200] + "..."
				}
				LogInfo("Meta entry %d: key='%s', value_preview='%s'", rowCount, entry.Key, valuePreview)
			}
		} else {
			LogWarn("Meta row %d has NULL value: key='%s'", rowCount, entry.Key)
		}
	}

	LogInfo("QueryMetaTable: queried %d rows, returned %d valid entries", rowCount, len(entries))

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return entries, nil
}

// BlobEntry represents an entry from the blobs table
type BlobEntry struct {
	Key   string
	Value string
}

// MetaEntry represents an entry from the meta table
type MetaEntry struct {
	Key   string
	Value string
}

// LoadSessionFromStoreDB loads session data from a single store.db file
func LoadSessionFromStoreDB(dbPath string) (map[string]*RawBubble, []*RawComposer, map[string][]*MessageContext, error) {
	db, err := OpenDatabase(dbPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to open store.db: %w", err)
	}
	defer db.Close()

	// Query both tables
	blobs, err := QueryBlobsTable(db)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to query blobs table: %w", err)
	}

	meta, err := QueryMetaTable(db)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to query meta table: %w", err)
	}

	// Extract session ID from path: ~/.cursor/chats/{hash}/{session-id}/store.db
	// Use this to help identify the session
	sessionID := extractSessionIDFromPath(dbPath)

	// Parse blobs and meta entries to extract bubbles, composers, and contexts
	bubbles := make(map[string]*RawBubble)
	var composers []*RawComposer
	contexts := make(map[string][]*MessageContext)

	// Process blobs - they may contain bubble data
	jsonParseFailures := 0
	for i, blob := range blobs {
		// Try to parse as JSON and identify the type
		var data map[string]interface{}
		valueBytes := []byte(blob.Value)

		// Try JSON first
		if err := json.Unmarshal(valueBytes, &data); err != nil {
			// Not JSON - try base64 decode in case it's encoded
			decoded, decodeErr := tryBase64Decode(blob.Value)
			if decodeErr == nil {
				if jsonErr := json.Unmarshal(decoded, &data); jsonErr == nil {
					// Successfully decoded and parsed
					LogInfo("Blob %d (key='%s') was base64 encoded, decoded successfully", i+1, blob.Key)
				} else {
					// Base64 decoded but not JSON - try extracting JSON from binary
					jsonBytes, found := extractJSONFromBinary(decoded)
					if found {
						if jsonErr := json.Unmarshal(jsonBytes, &data); jsonErr == nil {
							LogInfo("Blob %d (key='%s') had JSON embedded in binary data, extracted successfully", i+1, blob.Key)
						} else {
							jsonParseFailures++
							if i < 5 {
								valuePreview := blob.Value
								if len(valuePreview) > 100 {
									valuePreview = valuePreview[:100] + "..."
								}
								LogWarn("Blob %d (key='%s') failed JSON parse (tried base64 and binary extraction): %v. Value preview: %s", i+1, blob.Key, jsonErr, valuePreview)
							}
							continue
						}
					} else {
						// Decoded but still not JSON - log and skip
						jsonParseFailures++
						if i < 5 {
							valuePreview := blob.Value
							if len(valuePreview) > 100 {
								valuePreview = valuePreview[:100] + "..."
							}
							LogWarn("Blob %d (key='%s') failed JSON parse (tried base64 too): %v. Value preview: %s", i+1, blob.Key, jsonErr, valuePreview)
						}
						continue
					}
				}
			} else {
				// Not base64 - try extracting JSON from binary data
				jsonBytes, found := extractJSONFromBinary(valueBytes)
				if found {
					jsonPreview := string(jsonBytes)
					if len(jsonPreview) > 200 {
						jsonPreview = jsonPreview[:200] + "..."
					}
					LogInfo("Blob %d (key='%s'): Found JSON in binary (len=%d): %s", i+1, blob.Key, len(jsonBytes), jsonPreview)
					if jsonErr := json.Unmarshal(jsonBytes, &data); jsonErr == nil {
						LogInfo("Blob %d (key='%s') had JSON embedded in binary data, extracted successfully", i+1, blob.Key)
						// Log fields to understand structure
						keys := make([]string, 0, len(data))
						for k := range data {
							keys = append(keys, k)
						}
						LogInfo("Blob %d extracted JSON fields: %v", i+1, keys)
					} else {
						jsonParseFailures++
						if i < 10 {
							LogWarn("Blob %d (key='%s', key_len=%d) failed JSON parse (tried binary extraction): %v", i+1, blob.Key, len(blob.Key), jsonErr)
							LogInfo("  Extracted JSON preview: %s", jsonPreview)
						}
						continue
					}
				} else {
					// Not base64 and no JSON in binary - try parsing as text message format (text$uuid)
					// This handles cursor-agent's user message format: "hello$027f8b2f-d09c-4a69-98b0-b53f0118605d"
					if bubble := parseTextMessageFormat(blob.Key, blob.Value, sessionID); bubble != nil {
						bubbles[bubble.BubbleID] = bubble
						LogInfo("Blob %d parsed as text message format (user message): bubbleId='%s', text='%s', chatId='%s'", i+1, bubble.BubbleID, bubble.Text, bubble.ChatID)
						continue
					} else {
						// Log that we tried but failed to parse as text format
						if i < 5 {
							valuePreview := blob.Value
							if len(valuePreview) > 100 {
								valuePreview = valuePreview[:100] + "..."
							}
							LogInfo("Blob %d: tried text message format but didn't match pattern. Value preview: %s", i+1, valuePreview)
						}
					}

					// Not a text message format - the value might be a reference or in a different format
					// Log detailed info for first few failures to understand the format
					jsonParseFailures++
					if i < 10 {
						valuePreview := blob.Value
						fullValue := blob.Value
						if len(valuePreview) > 200 {
							valuePreview = valuePreview[:200] + "..."
						}
						LogWarn("Blob %d (key='%s', key_len=%d) failed JSON parse: %v", i+1, blob.Key, len(blob.Key), err)
						LogInfo("  Value (len=%d): %s", len(fullValue), valuePreview)
						LogInfo("  Key looks like hash: %v", isHashLike(blob.Key))
						// Check if value looks like a path or reference
						if strings.HasPrefix(fullValue, "/") || strings.Contains(fullValue, "$") {
							LogInfo("  Value appears to be a path/reference, not JSON data")
						}
						// Check if there's a { in the value that might indicate JSON
						if bytes.Contains(valueBytes, []byte("{")) {
							LogInfo("  Value contains '{' but extraction failed - JSON might be incomplete or malformed")
						}
					}
					continue
				}
			}
		}

		// Log available fields for all successfully parsed entries to understand structure
		keys := make([]string, 0, len(data))
		for k := range data {
			keys = append(keys, k)
		}
		LogInfo("Blob %d (key='%s') parsed successfully. Available fields: %v", i+1, blob.Key, keys)

		// Check if it's a bubble (has bubbleId)
		if _, ok := data["bubbleId"].(string); ok {
			bubble, err := parseBubbleFromData(blob.Key, data, sessionID)
			if err == nil {
				bubbles[bubble.BubbleID] = bubble
			}
		} else if id, ok := data["id"].(string); ok {
			// Check if it's a message format (has id, role, content) - cursor-agent format
			if role, hasRole := data["role"].(string); hasRole {
				bubble, err := parseMessageToBubble(blob.Key, id, role, data, sessionID)
				if err == nil {
					bubbles[bubble.BubbleID] = bubble
					LogInfo("Blob %d converted message (id='%s', role='%s') to bubble (bubbleId='%s')", i+1, id, role, bubble.BubbleID)
				} else {
					LogWarn("Blob %d failed to convert message to bubble: %v", i+1, err)
				}
			}
		}

		// Check if it's a composer (has composerId)
		if composerID, ok := data["composerId"].(string); ok {
			composer, err := parseComposerFromData(blob.Key, data)
			if err != nil {
				LogWarn("Failed to parse composer from blob key %s: %v", blob.Key, err)
				continue
			}
			if composer.ComposerID == "" {
				LogWarn("Composer parsed but missing composerId. Blob key: %s", blob.Key)
				continue
			}
			composer.ComposerID = composerID
			headerCount := len(composer.FullConversationHeadersOnly)
			LogInfo("Parsed composer %s: %d headers, name='%s'", composer.ComposerID, headerCount, composer.Name)
			composers = append(composers, composer)
		}
	}

	if jsonParseFailures > 0 {
		LogWarn("Failed to parse %d/%d blobs as JSON", jsonParseFailures, len(blobs))
	}

	// Process meta - may contain context or additional metadata
	metaJsonParseFailures := 0
	for i, entry := range meta {
		var data map[string]interface{}
		valueBytes := []byte(entry.Value)

		// Try JSON first
		if err := json.Unmarshal(valueBytes, &data); err != nil {
			// Not JSON - try base64 decode
			decoded, decodeErr := tryBase64Decode(entry.Value)
			if decodeErr == nil {
				if jsonErr := json.Unmarshal(decoded, &data); jsonErr == nil {
					LogInfo("Meta %d (key='%s') was base64 encoded, decoded successfully", i+1, entry.Key)
				} else {
					// Base64 decoded but not JSON - try hex decode
					hexDecoded, hexErr := tryHexDecode(entry.Value)
					if hexErr == nil {
						if jsonErr := json.Unmarshal(hexDecoded, &data); jsonErr == nil {
							LogInfo("Meta %d (key='%s') was hex encoded, decoded successfully", i+1, entry.Key)
						} else {
							metaJsonParseFailures++
							if i < 5 {
								valuePreview := entry.Value
								if len(valuePreview) > 100 {
									valuePreview = valuePreview[:100] + "..."
								}
								LogWarn("Meta %d (key='%s') failed JSON parse (tried base64 and hex): %v. Value preview: %s", i+1, entry.Key, jsonErr, valuePreview)
							}
							continue
						}
					} else {
						metaJsonParseFailures++
						if i < 5 {
							valuePreview := entry.Value
							if len(valuePreview) > 100 {
								valuePreview = valuePreview[:100] + "..."
							}
							LogWarn("Meta %d (key='%s') failed JSON parse (tried base64 too): %v. Value preview: %s", i+1, entry.Key, jsonErr, valuePreview)
						}
						continue
					}
				}
			} else {
				// Not base64 - try hex decode
				hexDecoded, hexErr := tryHexDecode(entry.Value)
				if hexErr == nil {
					if jsonErr := json.Unmarshal(hexDecoded, &data); jsonErr == nil {
						LogInfo("Meta %d (key='%s') was hex encoded, decoded successfully", i+1, entry.Key)
					} else {
						metaJsonParseFailures++
						if i < 10 {
							valuePreview := entry.Value
							fullValue := entry.Value
							if len(valuePreview) > 200 {
								valuePreview = valuePreview[:200] + "..."
							}
							LogWarn("Meta %d (key='%s', key_len=%d) failed JSON parse (tried hex): %v", i+1, entry.Key, len(entry.Key), jsonErr)
							LogInfo("  Value (len=%d): %s", len(fullValue), valuePreview)
						}
						continue
					}
				} else {
					metaJsonParseFailures++
					if i < 10 {
						valuePreview := entry.Value
						fullValue := entry.Value
						if len(valuePreview) > 200 {
							valuePreview = valuePreview[:200] + "..."
						}
						LogWarn("Meta %d (key='%s', key_len=%d) failed JSON parse: %v", i+1, entry.Key, len(entry.Key), err)
						LogInfo("  Value (len=%d): %s", len(fullValue), valuePreview)
						if strings.HasPrefix(fullValue, "/") || strings.Contains(fullValue, "$") {
							LogInfo("  Value appears to be a path/reference, not JSON data")
						}
					}
					continue
				}
			}
		}

		// Log available fields for first few entries
		if i < 3 {
			keys := make([]string, 0, len(data))
			for k := range data {
				keys = append(keys, k)
			}
			LogInfo("Meta %d (key='%s') parsed successfully. Available fields: %v", i+1, entry.Key, keys)
		}

		// Check if it's a message context
		if _, ok := data["contextId"].(string); ok {
			context, err := parseContextFromData(entry.Key, data)
			if err == nil {
				composerID := context.ComposerID
				if composerID == "" {
					// Try to extract from key or data
					if cid, ok := data["composerId"].(string); ok {
						composerID = cid
						context.ComposerID = composerID
					}
				}
				if composerID != "" {
					contexts[composerID] = append(contexts[composerID], context)
				}
			}
		}
	}

	if metaJsonParseFailures > 0 {
		LogWarn("Failed to parse %d/%d meta entries as JSON", metaJsonParseFailures, len(meta))
	}

	LogInfo("LoadSessionFromStoreDB summary: %d blobs queried, %d meta queried, %d bubbles extracted, %d composers extracted, %d contexts extracted",
		len(blobs), len(meta), len(bubbles), len(composers), len(contexts))

	return bubbles, composers, contexts, nil
}

// LoadAllSessionsFromAgentStorage loads all sessions from all store.db files
func (r *AgentStorageReader) LoadAllSessionsFromAgentStorage() (map[string]*RawBubble, []*RawComposer, map[string][]*MessageContext, error) {
	allBubbles := make(map[string]*RawBubble)
	var allComposers []*RawComposer
	allContexts := make(map[string][]*MessageContext)

	for _, dbPath := range r.storeDBPaths {
		bubbles, composers, contexts, err := LoadSessionFromStoreDB(dbPath)
		if err != nil {
			// Log error but continue with other files
			LogWarn("Failed to load session from %s: %v", dbPath, err)
			continue
		}

		// Merge bubbles (use bubbleID as key, so duplicates are overwritten)
		for id, bubble := range bubbles {
			allBubbles[id] = bubble
		}

		// Append composers
		allComposers = append(allComposers, composers...)
		LogInfo("Loaded from %s: %d bubbles, %d composers, %d context entries", dbPath, len(bubbles), len(composers), len(contexts))

		// Merge contexts
		for composerID, ctxList := range contexts {
			allContexts[composerID] = append(allContexts[composerID], ctxList...)
		}
	}

	LogInfo("Total loaded from agent storage: %d bubbles, %d composers, %d context groups", len(allBubbles), len(allComposers), len(allContexts))
	return allBubbles, allComposers, allContexts, nil
}

// Helper functions

func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func extractSessionIDFromPath(path string) string {
	// Extract session ID from path: ~/.cursor/chats/{hash}/{session-id}/store.db
	dir := filepath.Dir(path)
	sessionID := filepath.Base(dir)
	return sessionID
}

func parseBubbleFromData(key string, data map[string]interface{}, sessionID string) (*RawBubble, error) {
	bubble := &RawBubble{}

	// Extract bubbleId
	if id, ok := data["bubbleId"].(string); ok {
		bubble.BubbleID = id
	} else {
		return nil, fmt.Errorf("missing bubbleId in data")
	}

	// Extract chatId (use sessionID if not present)
	if chatID, ok := data["chatId"].(string); ok {
		bubble.ChatID = chatID
	} else {
		bubble.ChatID = sessionID
	}

	// Extract text
	if text, ok := data["text"].(string); ok {
		bubble.Text = text
	}

	// Extract richText
	if richText, ok := data["richText"].(string); ok {
		bubble.RichText = richText
	}

	// Extract codeBlocks
	if codeBlocks, ok := data["codeBlocks"].([]interface{}); ok {
		for _, cb := range codeBlocks {
			if cbMap, ok := cb.(map[string]interface{}); ok {
				codeBlock := CodeBlock{}
				if lang, ok := cbMap["language"].(string); ok {
					codeBlock.Language = lang
				}
				if content, ok := cbMap["content"].(string); ok {
					codeBlock.Content = content
				}
				bubble.CodeBlocks = append(bubble.CodeBlocks, codeBlock)
			}
		}
	}

	// Extract timestamp
	if ts, ok := data["timestamp"].(float64); ok {
		bubble.Timestamp = int64(ts)
	} else if ts, ok := data["timestamp"].(int64); ok {
		bubble.Timestamp = ts
	}

	// Extract type
	if t, ok := data["type"].(float64); ok {
		bubble.Type = int(t)
	} else if t, ok := data["type"].(int); ok {
		bubble.Type = t
	}

	return bubble, nil
}

// parseTextMessageFormat parses cursor-agent's text message format: "text$uuid"
// Returns a RawBubble if the format matches, nil otherwise
// Handles format like: "hello$027f8b2f-d09c-4a69-98b0-b53f0118605d" (may have control chars)
func parseTextMessageFormat(key, value, sessionID string) *RawBubble {
	// First, aggressively remove all control characters except newlines/tabs/carriage returns
	// This handles cases where the value starts with control chars like \x05, \n, etc.
	cleaned := strings.Map(func(r rune) rune {
		// Keep printable characters, newlines, tabs, carriage returns, and space
		if r >= 32 || r == '\n' || r == '\r' || r == '\t' {
			return r
		}
		// Remove all other control characters
		return -1
	}, value)

	// Trim whitespace from both ends
	cleaned = strings.TrimSpace(cleaned)

	// Check if value matches pattern: text$uuid
	// Example: "hello$027f8b2f-d09c-4a69-98b0-b53f0118605d"
	dollarIdx := strings.Index(cleaned, "$")
	if dollarIdx == -1 || dollarIdx == 0 {
		return nil // No $ found or $ is at start
	}

	// Extract text before $ and clean it
	text := strings.TrimSpace(cleaned[:dollarIdx])
	// Remove any remaining control characters (shouldn't be any after first pass, but be safe)
	text = strings.Map(func(r rune) rune {
		if r < 32 && r != '\n' && r != '\r' && r != '\t' {
			return -1 // Remove control characters except newlines/tabs
		}
		return r
	}, text)
	text = strings.TrimSpace(text)

	if text == "" {
		return nil // No text before $
	}

	// Extract UUID after $ (optional, but useful for bubble ID)
	uuidPart := ""
	if dollarIdx+1 < len(cleaned) {
		uuidPart = strings.TrimSpace(cleaned[dollarIdx+1:])
		// Remove control characters from UUID
		uuidPart = strings.Map(func(r rune) rune {
			if r < 32 {
				return -1
			}
			return r
		}, uuidPart)
	}

	// Use UUID as bubble ID if available, otherwise use a hash of the text
	bubbleID := uuidPart
	if bubbleID == "" {
		// Generate a simple ID from the blob key (first 8 chars)
		if len(key) >= 8 {
			bubbleID = key[:8]
		} else {
			bubbleID = key
		}
	}

	// Create user bubble (type=1 for user messages)
	bubble := &RawBubble{
		BubbleID:  bubbleID,
		ChatID:    sessionID,
		Type:      1, // User message
		Text:      text,
		Timestamp: time.Now().UnixMilli(), // Use current time if not available
	}

	return bubble
}

// parseMessageToBubble converts a message format (id, role, content) to a RawBubble
// This handles cursor-agent's message format where messages have id, role, and content fields
func parseMessageToBubble(key, id, role string, data map[string]interface{}, sessionID string) (*RawBubble, error) {
	bubble := &RawBubble{
		BubbleID: id,
		ChatID:   sessionID,
	}

	// Map role to type: "user" = 1, "assistant" = 2
	if role == "user" {
		bubble.Type = 1
	} else if role == "assistant" {
		bubble.Type = 2
	} else {
		// Default to assistant if unknown
		bubble.Type = 2
	}

	// Extract text from content array
	if content, ok := data["content"].([]interface{}); ok {
		var textParts []string
		for _, item := range content {
			if itemMap, ok := item.(map[string]interface{}); ok {
				// Check for text content
				if text, ok := itemMap["text"].(string); ok {
					textParts = append(textParts, text)
				} else if data, ok := itemMap["data"].(string); ok {
					// Some content items have "data" field (like redacted-reasoning)
					// We can skip these or add them as metadata
					// For now, skip redacted content
					if itemType, _ := itemMap["type"].(string); itemType != "redacted-reasoning" {
						textParts = append(textParts, data)
					}
				}
			}
		}
		if len(textParts) > 0 {
			bubble.Text = strings.Join(textParts, "\n\n")
		}
	}

	// Extract timestamp if available
	if ts, ok := data["timestamp"].(float64); ok {
		bubble.Timestamp = int64(ts)
	} else if ts, ok := data["timestamp"].(int64); ok {
		bubble.Timestamp = ts
	} else {
		// Use current time if no timestamp
		bubble.Timestamp = time.Now().Unix()
	}

	return bubble, nil
}

func parseComposerFromData(key string, data map[string]interface{}) (*RawComposer, error) {
	composer := &RawComposer{}

	// Extract composerId
	if id, ok := data["composerId"].(string); ok {
		composer.ComposerID = id
	}

	// Extract name
	if name, ok := data["name"].(string); ok {
		composer.Name = name
	}

	// Extract fullConversationHeadersOnly
	if headers, ok := data["fullConversationHeadersOnly"].([]interface{}); ok {
		for _, h := range headers {
			if hMap, ok := h.(map[string]interface{}); ok {
				header := ConversationHeader{}
				if bubbleID, ok := hMap["bubbleId"].(string); ok {
					header.BubbleID = bubbleID
				}
				if t, ok := hMap["type"].(float64); ok {
					header.Type = int(t)
				} else if t, ok := hMap["type"].(int); ok {
					header.Type = t
				}
				composer.FullConversationHeadersOnly = append(composer.FullConversationHeadersOnly, header)
			}
		}
	}

	// Fallback to legacy format: conversation[] array
	if len(composer.FullConversationHeadersOnly) == 0 {
		// Try legacy format: conversation[] array
		if convArray, ok := data["conversation"].([]interface{}); ok && len(convArray) > 0 {
			LogInfo("Composer %s: Using legacy conversation[] format (found %d entries)", composer.ComposerID, len(convArray))
			// Convert legacy format to headers
			for _, entry := range convArray {
				if entryMap, ok := entry.(map[string]interface{}); ok {
					header := ConversationHeader{}
					if bubbleID, ok := entryMap["bubbleId"].(string); ok {
						header.BubbleID = bubbleID
					}
					if t, ok := entryMap["type"].(float64); ok {
						header.Type = int(t)
					} else if t, ok := entryMap["type"].(int); ok {
						header.Type = t
					}
					if header.BubbleID != "" {
						composer.FullConversationHeadersOnly = append(composer.FullConversationHeadersOnly, header)
					}
				}
			}
		} else {
			// Log available fields for debugging
			keys := make([]string, 0, len(data))
			for k := range data {
				keys = append(keys, k)
			}
			LogWarn("Composer %s: No conversation data found. Available fields: %v", composer.ComposerID, keys)
		}
	}

	// Extract timestamps
	if ts, ok := data["createdAt"].(float64); ok {
		composer.CreatedAt = int64(ts)
	} else if ts, ok := data["createdAt"].(int64); ok {
		composer.CreatedAt = ts
	}

	if ts, ok := data["lastUpdatedAt"].(float64); ok {
		composer.LastUpdatedAt = int64(ts)
	} else if ts, ok := data["lastUpdatedAt"].(int64); ok {
		composer.LastUpdatedAt = ts
	}

	return composer, nil
}

func parseContextFromData(key string, data map[string]interface{}) (*MessageContext, error) {
	context := &MessageContext{}

	// Extract contextId
	if id, ok := data["contextId"].(string); ok {
		context.ContextID = id
	}

	// Extract bubbleId
	if id, ok := data["bubbleId"].(string); ok {
		context.BubbleID = id
	}

	// Extract composerId
	if id, ok := data["composerId"].(string); ok {
		context.ComposerID = id
	}

	// Extract other optional fields
	if gitStatus, ok := data["gitStatusRaw"].(string); ok {
		context.GitStatusRaw = gitStatus
	}

	if terminalFiles, ok := data["terminalFiles"].([]interface{}); ok {
		for _, tf := range terminalFiles {
			if str, ok := tf.(string); ok {
				context.TerminalFiles = append(context.TerminalFiles, str)
			}
		}
	}

	if projectLayouts, ok := data["projectLayouts"].([]interface{}); ok {
		for _, pl := range projectLayouts {
			if str, ok := pl.(string); ok {
				context.ProjectLayouts = append(context.ProjectLayouts, str)
			}
		}
	}

	return context, nil
}

// tryBase64Decode attempts to decode a base64 string, returns decoded bytes or error
func tryBase64Decode(s string) ([]byte, error) {
	// Try standard base64
	decoded, err := base64.StdEncoding.DecodeString(s)
	if err == nil {
		return decoded, nil
	}
	// Try URL-safe base64
	decoded, err = base64.URLEncoding.DecodeString(s)
	if err == nil {
		return decoded, nil
	}
	// Try with padding
	if len(s)%4 != 0 {
		padded := s + strings.Repeat("=", 4-len(s)%4)
		decoded, err = base64.StdEncoding.DecodeString(padded)
		if err == nil {
			return decoded, nil
		}
	}
	return nil, fmt.Errorf("not base64 encoded")
}

// tryHexDecode attempts to decode a hex-encoded string, returns decoded bytes or error
func tryHexDecode(s string) ([]byte, error) {
	// Remove whitespace, newlines, and tabs
	cleaned := strings.ReplaceAll(s, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "\n", "")
	cleaned = strings.ReplaceAll(cleaned, "\t", "")
	cleaned = strings.ReplaceAll(cleaned, "\r", "")

	decoded, err := hex.DecodeString(cleaned)
	if err != nil {
		return nil, fmt.Errorf("not hex encoded: %w", err)
	}
	return decoded, nil
}

// extractJSONFromBinary attempts to extract a JSON object from binary data
// Returns the JSON bytes and true if found, or nil and false if not found
func extractJSONFromBinary(data []byte) ([]byte, bool) {
	// Look for JSON object start
	startIdx := bytes.Index(data, []byte("{"))
	if startIdx == -1 {
		return nil, false
	}

	// Try to find matching closing brace with proper brace counting
	// Need to handle strings that might contain braces
	depth := 0
	inString := false
	escapeNext := false

	for i := startIdx; i < len(data); i++ {
		if escapeNext {
			escapeNext = false
			continue
		}

		if data[i] == '\\' {
			escapeNext = true
			continue
		}

		if data[i] == '"' && !escapeNext {
			inString = !inString
			continue
		}

		if !inString {
			if data[i] == '{' {
				depth++
			} else if data[i] == '}' {
				depth--
				if depth == 0 {
					// Found complete JSON object
					return data[startIdx : i+1], true
				}
			}
		}
	}

	return nil, false
}

// isHashLike checks if a string looks like a hash (hex characters, reasonable length)
func isHashLike(s string) bool {
	if len(s) < 16 || len(s) > 128 {
		return false
	}
	for _, r := range s {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
}
