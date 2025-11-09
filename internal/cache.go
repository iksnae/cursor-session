package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// CacheManager handles caching of reconstructed conversations
type CacheManager struct {
	cacheDir string
}

// CacheMetadata stores metadata about the cache
type CacheMetadata struct {
	DatabasePath    string    `json:"database_path" yaml:"database_path"`
	DatabaseModTime time.Time `json:"database_mod_time" yaml:"database_mod_time"`
	CacheVersion    string    `json:"cache_version" yaml:"cache_version"`
	CreatedAt       time.Time `json:"created_at" yaml:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" yaml:"updated_at"`
}

// SessionIndexEntry represents a session entry in the index
type SessionIndexEntry struct {
	ID           string `yaml:"id"`
	ComposerID   string `yaml:"composer_id"`
	Name         string `yaml:"name,omitempty"`
	CreatedAt    string `yaml:"created_at,omitempty"`
	UpdatedAt    string `yaml:"updated_at,omitempty"`
	MessageCount int    `yaml:"message_count"`
	Workspace    string `yaml:"workspace,omitempty"`
}

// SessionIndex represents the YAML index of all sessions
type SessionIndex struct {
	Sessions []SessionIndexEntry `yaml:"sessions"`
	Metadata CacheMetadata       `yaml:"metadata"`
}

// NewCacheManager creates a new cache manager
func NewCacheManager(cacheDir string) *CacheManager {
	return &CacheManager{
		cacheDir: cacheDir,
	}
}

// EnsureCacheDir ensures the cache directory exists
func (cm *CacheManager) EnsureCacheDir() error {
	return os.MkdirAll(cm.cacheDir, 0755)
}

// GetIndexPath returns the path to the session index YAML file
func (cm *CacheManager) GetIndexPath() string {
	return filepath.Join(cm.cacheDir, "sessions.yaml")
}

// GetSessionPath returns the path to a session's cache file
func (cm *CacheManager) GetSessionPath(sessionID string) string {
	return filepath.Join(cm.cacheDir, fmt.Sprintf("session_%s.json", sessionID))
}

// IsCacheValid checks if the cache is valid for the given database
func (cm *CacheManager) IsCacheValid(dbPath string) (bool, error) {
	indexPath := cm.GetIndexPath()

	// Check if index exists
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		return false, nil
	}

	// Load index
	index, err := cm.LoadIndex()
	if err != nil {
		return false, nil
	}

	// Check if database path matches
	if index.Metadata.DatabasePath != dbPath {
		return false, nil
	}

	// Check if database modification time matches
	dbInfo, err := os.Stat(dbPath)
	if err != nil {
		return false, nil
	}

	if !index.Metadata.DatabaseModTime.Equal(dbInfo.ModTime()) {
		return false, nil
	}

	return true, nil
}

// GetCacheDir returns the cache directory path
func (cm *CacheManager) GetCacheDir() string {
	return cm.cacheDir
}

// LoadIndex loads the session index
func (cm *CacheManager) LoadIndex() (*SessionIndex, error) {
	indexPath := cm.GetIndexPath()
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return nil, err
	}

	var index SessionIndex
	if err := yaml.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("failed to unmarshal index: %w", err)
	}

	return &index, nil
}

// SaveIndex saves the session index
func (cm *CacheManager) SaveIndex(index *SessionIndex) error {
	if err := cm.EnsureCacheDir(); err != nil {
		return err
	}

	indexPath := cm.GetIndexPath()
	data, err := yaml.Marshal(index)
	if err != nil {
		return fmt.Errorf("failed to marshal index: %w", err)
	}

	return os.WriteFile(indexPath, data, 0644)
}

// SaveSession saves a single session to its cache file
func (cm *CacheManager) SaveSession(session *Session) error {
	if err := cm.EnsureCacheDir(); err != nil {
		return err
	}

	sessionPath := cm.GetSessionPath(session.ID)
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	return os.WriteFile(sessionPath, data, 0644)
}

// LoadSession loads a single session from its cache file
func (cm *CacheManager) LoadSession(sessionID string) (*Session, error) {
	sessionPath := cm.GetSessionPath(sessionID)
	data, err := os.ReadFile(sessionPath)
	if err != nil {
		return nil, err
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &session, nil
}

// LoadAllSessions loads all sessions from cache
func (cm *CacheManager) LoadAllSessions() ([]*Session, error) {
	index, err := cm.LoadIndex()
	if err != nil {
		return nil, err
	}

	var sessions []*Session
	for _, entry := range index.Sessions {
		session, err := cm.LoadSession(entry.ID)
		if err != nil {
			// Log but continue
			continue
		}
		sessions = append(sessions, session)
	}

	return sessions, nil
}

// SaveSessionAndUpdateIndex saves a single session and updates the index
func (cm *CacheManager) SaveSessionAndUpdateIndex(session *Session, dbPath string) error {
	if err := cm.EnsureCacheDir(); err != nil {
		return err
	}

	dbInfo, err := os.Stat(dbPath)
	if err != nil {
		return err
	}

	// Load existing index or create new one
	var index *SessionIndex
	existingIndex, err := cm.LoadIndex()
	if err == nil && existingIndex != nil {
		// Check if index is valid for this database
		if existingIndex.Metadata.DatabasePath == dbPath {
			index = existingIndex
			// Update metadata to reflect current database state
			index.Metadata.DatabaseModTime = dbInfo.ModTime()
			index.Metadata.UpdatedAt = time.Now()
		}
	}

	// Create new index if needed
	if index == nil {
		index = &SessionIndex{
			Sessions: make([]SessionIndexEntry, 0),
			Metadata: CacheMetadata{
				DatabasePath:    dbPath,
				DatabaseModTime: dbInfo.ModTime(),
				CacheVersion:    "1.0",
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			},
		}
	}

	// Save session file
	if err := cm.SaveSession(session); err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	// Update or add session entry in index
	found := false
	for i, entry := range index.Sessions {
		if entry.ComposerID == session.Metadata.ComposerID {
			// Update existing entry
			index.Sessions[i] = SessionIndexEntry{
				ID:           session.ID,
				ComposerID:   session.Metadata.ComposerID,
				Name:         session.Metadata.Name,
				CreatedAt:    session.Metadata.CreatedAt,
				UpdatedAt:    session.Metadata.UpdatedAt,
				MessageCount: len(session.Messages),
				Workspace:    session.Workspace,
			}
			found = true
			break
		}
	}

	if !found {
		// Add new entry
		index.Sessions = append(index.Sessions, SessionIndexEntry{
			ID:           session.ID,
			ComposerID:   session.Metadata.ComposerID,
			Name:         session.Metadata.Name,
			CreatedAt:    session.Metadata.CreatedAt,
			UpdatedAt:    session.Metadata.UpdatedAt,
			MessageCount: len(session.Messages),
			Workspace:    session.Workspace,
		})
	}

	// Save updated index
	return cm.SaveIndex(index)
}

// SaveSessions saves all sessions and updates the index
func (cm *CacheManager) SaveSessions(sessions []*Session, dbPath string) error {
	if err := cm.EnsureCacheDir(); err != nil {
		return err
	}

	dbInfo, err := os.Stat(dbPath)
	if err != nil {
		return err
	}

	// Build index
	index := SessionIndex{
		Sessions: make([]SessionIndexEntry, 0, len(sessions)),
		Metadata: CacheMetadata{
			DatabasePath:    dbPath,
			DatabaseModTime: dbInfo.ModTime(),
			CacheVersion:    "1.0",
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
	}

	// Save each session and add to index
	for _, session := range sessions {
		if err := cm.SaveSession(session); err != nil {
			LogWarn("Failed to save session %s: %v", session.ID, err)
			continue
		}

		index.Sessions = append(index.Sessions, SessionIndexEntry{
			ID:           session.ID,
			ComposerID:   session.Metadata.ComposerID,
			Name:         session.Metadata.Name,
			CreatedAt:    session.Metadata.CreatedAt,
			UpdatedAt:    session.Metadata.UpdatedAt,
			MessageCount: len(session.Messages),
			Workspace:    session.Workspace,
		})
	}

	// Save index
	return cm.SaveIndex(&index)
}

// LoadConversations loads reconstructed conversations from cache (for backward compatibility)
// Note: This is a simplified conversion and may lose some data
func (cm *CacheManager) LoadConversations() ([]*ReconstructedConversation, error) {
	sessions, err := cm.LoadAllSessions()
	if err != nil {
		return nil, err
	}

	// Convert sessions back to conversations (simplified)
	// This is a lossy conversion, but needed for compatibility
	conversations := make([]*ReconstructedConversation, 0, len(sessions))
	for _, session := range sessions {
		conv := &ReconstructedConversation{
			ComposerID: session.Metadata.ComposerID,
			Name:       session.Metadata.Name,
			CreatedAt:  parseTimestamp(session.Metadata.CreatedAt),
			UpdatedAt:  parseTimestamp(session.Metadata.UpdatedAt),
			Messages:   make([]ReconstructedMessage, 0, len(session.Messages)),
		}

		// Convert messages
		for _, msg := range session.Messages {
			msgType := 2 // default to assistant
			if msg.Actor == "user" {
				msgType = 1
			}
			reconstructedMsg := ReconstructedMessage{
				BubbleID:  fmt.Sprintf("bubble_%d", len(conv.Messages)),
				Text:      msg.Content,
				Type:      msgType,
				Timestamp: parseTimestamp(msg.Timestamp),
			}
			conv.Messages = append(conv.Messages, reconstructedMsg)
		}

		conversations = append(conversations, conv)
	}

	return conversations, nil
}

// ClearCache clears the cache
func (cm *CacheManager) ClearCache() error {
	indexPath := cm.GetIndexPath()

	// Load index to get all session IDs
	index, err := cm.LoadIndex()
	if err == nil {
		// Delete all session files
		for _, entry := range index.Sessions {
			sessionPath := cm.GetSessionPath(entry.ID)
			_ = os.Remove(sessionPath)
		}
	}

	// Delete index
	if err := os.Remove(indexPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

// parseTimestamp parses a timestamp string to int64
func parseTimestamp(ts string) int64 {
	if ts == "" {
		return 0
	}
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return 0
	}
	return t.UnixMilli()
}
