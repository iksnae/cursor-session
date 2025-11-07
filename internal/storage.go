package internal

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

// Storage provides methods to extract raw data from cursorDiskKV
type Storage struct {
	db *sql.DB
}

// NewStorage creates a new Storage instance
func NewStorage(db *sql.DB) *Storage {
	return &Storage{db: db}
}

// LoadBubbles loads all bubbles from the database
func (s *Storage) LoadBubbles() (map[string]*RawBubble, error) {
	pairs, err := QueryCursorDiskKV(s.db, "bubbleId:%")
	if err != nil {
		return nil, fmt.Errorf("failed to query bubbles: %w", err)
	}

	bubbleMap := make(map[string]*RawBubble)
	for _, pair := range pairs {
		bubble, err := ParseRawBubble(pair.Key, pair.Value)
		if err != nil {
			// Log error but continue
			continue
		}
		// Use bubbleId as key for lookup
		bubbleMap[bubble.BubbleID] = bubble
	}

	return bubbleMap, nil
}

// LoadComposers loads all composers from the database
func (s *Storage) LoadComposers() ([]*RawComposer, error) {
	pairs, err := QueryCursorDiskKV(s.db, "composerData:%")
	if err != nil {
		return nil, fmt.Errorf("failed to query composers: %w", err)
	}

	composers := make([]*RawComposer, 0)
	for _, pair := range pairs {
		composer, err := ParseRawComposer(pair.Key, pair.Value)
		if err != nil {
			// Log error but continue
			continue
		}
		composers = append(composers, composer)
	}

	return composers, nil
}

// LoadMessageContexts loads all message contexts from the database
func (s *Storage) LoadMessageContexts() (map[string][]*MessageContext, error) {
	pairs, err := QueryCursorDiskKV(s.db, "messageRequestContext:%")
	if err != nil {
		return nil, fmt.Errorf("failed to query message contexts: %w", err)
	}

	contextMap := make(map[string][]*MessageContext)
	for _, pair := range pairs {
		context, err := ParseMessageContext(pair.Key, pair.Value)
		if err != nil {
			// Log error but continue
			continue
		}
		// Group by composerId
		contextMap[context.ComposerID] = append(contextMap[context.ComposerID], context)
	}

	return contextMap, nil
}

// LoadCodeBlockDiffs loads all code block diffs from the database
func (s *Storage) LoadCodeBlockDiffs() (map[string][]interface{}, error) {
	pairs, err := QueryCursorDiskKV(s.db, "codeBlockDiff:%")
	if err != nil {
		return nil, fmt.Errorf("failed to query code block diffs: %w", err)
	}

	diffMap := make(map[string][]interface{})
	for _, pair := range pairs {
		// Extract chatId from key: codeBlockDiff:<chatId>:<diffId>
		parts := splitKey(pair.Key, "codeBlockDiff:")
		if len(parts) < 2 {
			continue
		}
		chatId := parts[1]

		var diff interface{}
		if err := json.Unmarshal([]byte(pair.Value), &diff); err != nil {
			continue
		}

		diffMap[chatId] = append(diffMap[chatId], diff)
	}

	return diffMap, nil
}
