package core

import "time"

// SessionMeta holds lightweight metadata about a persisted session,
// stored in the index file for fast listing without loading full messages.
type SessionMeta struct {
	ID           string `json:"id"`
	Profile      string `json:"profile"`
	Name         string `json:"name,omitempty"`
	Model        string `json:"model,omitempty"`
	Summary      string `json:"summary,omitempty"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at,omitempty"`
	MessageCount int    `json:"message_count,omitempty"`
}

// Session is a full conversation turn history with its metadata.
// Messages are stored as the core.Message type so they round-trip cleanly
// through JSON (see session_test.go TestMessageJSONRoundTrip).
type Session struct {
	Meta     SessionMeta `json:"meta"`
	Messages []Message   `json:"messages"`
}

// SessionStore is the port for session persistence (D-layer hexagonal port).
// Implementations live in the adapters layer (e.g. FileSessionStore).
// The core domain depends only on this interface — no I/O, no file paths.
type SessionStore interface {
	// Save persists a session (messages + metadata) atomically.
	Save(session *Session) error

	// Load retrieves a full session by ID. Returns an error if not found.
	Load(id string) (*Session, error)

	// List returns metadata for all stored sessions, newest first.
	List() ([]SessionMeta, error)

	// Delete removes a session by ID (file + index entry).
	Delete(id string) error

	// Rename sets a custom name on a session (updates file + index).
	Rename(id string, name string) error
}

// NewSessionMeta creates a SessionMeta with the current UTC timestamp.
func NewSessionMeta(id, profile string) SessionMeta {
	return SessionMeta{
		ID:        id,
		Profile:   profile,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
}
