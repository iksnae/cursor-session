package internal

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
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
	agentStorageChecked := false
	if paths.HasAgentStorage() {
		agentStorageChecked = true
		storeDBs, err := paths.FindAgentStoreDBs()
		if err != nil {
			// Log warning but continue to provide helpful error message
			LogWarn("Error scanning agent storage directory: %v", err)
			// Continue to show helpful error message below
		} else if len(storeDBs) > 0 {
			LogInfo("Found %d session database(s) in agent storage", len(storeDBs))
			return NewAgentStorage(storeDBs), nil
		} else {
			// Directory exists but no store.db files found
			LogInfo("Agent storage directory exists but no store.db files found")
		}
	}

	// Neither format available - provide detailed error message
	var errMsg strings.Builder
	errMsg.WriteString("no Cursor storage found\n\n")
	errMsg.WriteString("Checked storage locations:\n")
	errMsg.WriteString(fmt.Sprintf("  • Desktop app: %s (not found)\n", paths.GetGlobalStorageDBPath()))
	
	if agentStorageChecked {
		if paths.AgentStoragePath != "" {
			errMsg.WriteString(fmt.Sprintf("  • Agent CLI: %s (directory exists but no store.db files found)\n", paths.AgentStoragePath))
			errMsg.WriteString(fmt.Sprintf("    → Expected pattern: %s/{hash}/{session-id}/store.db\n", paths.AgentStoragePath))
			errMsg.WriteString(fmt.Sprintf("    → Sessions are created when cursor-agent CLI runs with chat interactions\n"))
		} else {
			errMsg.WriteString("  • Agent CLI: not available on this platform\n")
		}
	} else {
		if paths.AgentStoragePath != "" {
			errMsg.WriteString(fmt.Sprintf("  • Agent CLI: %s (directory not found)\n", paths.AgentStoragePath))
			errMsg.WriteString(fmt.Sprintf("    → This directory is created when cursor-agent CLI is first used\n"))
		} else {
			errMsg.WriteString("  • Agent CLI: not available on this platform\n")
		}
	}
	
	// Check if we're in a CI environment
	if IsCIEnvironment() {
		errMsg.WriteString("\n")
		errMsg.WriteString("CI/CD Environment Detected:\n")
		errMsg.WriteString("  • This is expected if cursor-agent hasn't created sessions yet.\n")
		errMsg.WriteString("  • Sessions are created automatically when cursor-agent CLI runs.\n")
		errMsg.WriteString("  • If you just ran cursor-agent commands, sessions should appear shortly.\n")
		errMsg.WriteString("  • Try running: ls -la ~/.cursor/chats/ to verify session directories exist.\n")
		errMsg.WriteString("  • Each session directory should contain a store.db file.\n")
	} else {
		errMsg.WriteString("\n")
		errMsg.WriteString("To use this tool, you need either:\n")
		errMsg.WriteString("  • Cursor IDE desktop app with chat history, or\n")
		errMsg.WriteString("  • cursor-agent CLI with active sessions in ~/.cursor/chats/\n")
	}
	
	return nil, fmt.Errorf("%s", errMsg.String())
}

// IsCIEnvironment checks if we're running in a CI/CD environment
func IsCIEnvironment() bool {
	// Check common CI environment variables
	ciVars := []string{
		"CI",                    // Generic CI indicator
		"GITHUB_ACTIONS",        // GitHub Actions
		"GITLAB_CI",             // GitLab CI
		"JENKINS_URL",           // Jenkins
		"CIRCLECI",              // CircleCI
		"TRAVIS",                // Travis CI
		"BUILDKITE",             // Buildkite
		"TEAMCITY_VERSION",      // TeamCity
		"TF_BUILD",              // Azure DevOps
		"bamboo_buildKey",       // Bamboo
	}
	
	for _, envVar := range ciVars {
		if os.Getenv(envVar) != "" {
			return true
		}
	}
	
	return false
}
