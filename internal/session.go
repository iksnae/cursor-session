package internal

// Session represents a normalized chat session
type Session struct {
	ID        string    `json:"id"`
	Workspace string    `json:"workspace,omitempty"`
	Source    string    `json:"source"` // "globalStorage"
	Messages  []Message `json:"messages"`
	Metadata  Metadata  `json:"metadata,omitempty"`
}

// Message represents a normalized message
type Message struct {
	Timestamp string `json:"timestamp,omitempty"`
	Actor     string `json:"actor"` // "user", "assistant", "tool"
	Content   string `json:"content"`
}

// Metadata contains additional session information
type Metadata struct {
	Key          string `json:"key,omitempty"`
	CreatedAt    string `json:"created_at,omitempty"`
	UpdatedAt    string `json:"updated_at,omitempty"`
	MessageCount int    `json:"message_count"`
	ComposerID   string `json:"composer_id,omitempty"`
	Name         string `json:"name,omitempty"`
}
