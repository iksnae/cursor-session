package internal

import (
	"crypto/sha256"
	"encoding/hex"
)

// Deduplicator removes duplicate sessions
type Deduplicator struct{}

// NewDeduplicator creates a new Deduplicator
func NewDeduplicator() *Deduplicator {
	return &Deduplicator{}
}

// Deduplicate removes duplicate sessions based on content hash
func (d *Deduplicator) Deduplicate(sessions []*Session) []*Session {
	seen := make(map[string]bool)
	var unique []*Session

	for _, session := range sessions {
		hash := d.hashSessionContent(session)
		if !seen[hash] {
			seen[hash] = true
			unique = append(unique, session)
		}
	}

	return unique
}

// hashSessionContent creates a content-based hash for a session
func (d *Deduplicator) hashSessionContent(session *Session) string {
	h := sha256.New()

	// Hash all message content
	for _, msg := range session.Messages {
		h.Write([]byte(msg.Actor))
		h.Write([]byte(msg.Content))
		h.Write([]byte(msg.Timestamp))
	}

	return hex.EncodeToString(h.Sum(nil))
}
