package internal

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

// StorageBackend is the interface for storage backends that can load session data
type StorageBackend interface {
	LoadBubbles() (map[string]*RawBubble, error)
	LoadComposers() ([]*RawComposer, error)
	LoadMessageContexts() (map[string][]*MessageContext, error)
	LoadCodeBlockDiffs() (map[string][]interface{}, error)
}

// Storage provides methods to extract raw data from cursorDiskKV (desktop app format)
type Storage struct {
	db *sql.DB
}

// Ensure Storage implements StorageBackend
var _ StorageBackend = (*Storage)(nil)

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

// AgentStorage provides methods to extract raw data from cursor-agent CLI store.db files
type AgentStorage struct {
	reader *AgentStorageReader
}

// NewAgentStorage creates a new AgentStorage instance
func NewAgentStorage(storeDBPaths []string) *AgentStorage {
	return &AgentStorage{
		reader: NewAgentStorageReader(storeDBPaths),
	}
}

// Ensure AgentStorage implements StorageBackend
var _ StorageBackend = (*AgentStorage)(nil)

// LoadBubbles loads all bubbles from agent storage
func (a *AgentStorage) LoadBubbles() (map[string]*RawBubble, error) {
	bubbles, _, _, err := a.reader.LoadAllSessionsFromAgentStorage()
	if err != nil {
		return nil, fmt.Errorf("failed to load bubbles from agent storage: %w", err)
	}
	return bubbles, nil
}

// LoadComposers loads all composers from agent storage
func (a *AgentStorage) LoadComposers() ([]*RawComposer, error) {
	_, composers, _, err := a.reader.LoadAllSessionsFromAgentStorage()
	if err != nil {
		return nil, fmt.Errorf("failed to load composers from agent storage: %w", err)
	}
	return composers, nil
}

// LoadMessageContexts loads all message contexts from agent storage
func (a *AgentStorage) LoadMessageContexts() (map[string][]*MessageContext, error) {
	_, _, contexts, err := a.reader.LoadAllSessionsFromAgentStorage()
	if err != nil {
		return nil, fmt.Errorf("failed to load contexts from agent storage: %w", err)
	}
	return contexts, nil
}

// LoadCodeBlockDiffs loads all code block diffs from agent storage
// Note: Agent storage format may not have code block diffs in the same way
// This returns an empty map for now, but can be extended if needed
func (a *AgentStorage) LoadCodeBlockDiffs() (map[string][]interface{}, error) {
	// Agent storage format doesn't currently support code block diffs
	// Return empty map to maintain interface compatibility
	return make(map[string][]interface{}), nil
}

// NewStorageBackend creates a StorageBackend based on available storage formats
// It prioritizes desktop app format (globalStorage) over agent storage
func NewStorageBackend(paths StoragePaths) (StorageBackend, error) {
	// First, try desktop app format (globalStorage)
	if paths.GlobalStorageExists() {
		dbPath := paths.GetGlobalStorageDBPath()
		db, err := OpenDatabase(dbPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open globalStorage database: %w", err)
		}
		return NewStorage(db), nil
	}

	// Fallback to agent storage if available
	if paths.HasAgentStorage() {
		storeDBs, err := paths.FindAgentStoreDBs()
		if err != nil {
			return nil, fmt.Errorf("failed to find agent store databases: %w", err)
		}
		if len(storeDBs) > 0 {
			return NewAgentStorage(storeDBs), nil
		}
	}

	// Neither format available
	return nil, fmt.Errorf("no storage format available: globalStorage not found at %s, agent storage not found at %s", paths.GetGlobalStorageDBPath(), paths.AgentStoragePath)
}
