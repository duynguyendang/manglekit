package core

import (
	"context"
)

// MemoryMode defines the persistence strategy for conversation state.
type MemoryMode string

const (
	// MemoryModeNone indicates no state persistence (stateless execution).
	MemoryModeNone MemoryMode = "none"
	// MemoryModeTransient indicates in-memory persistence that lasts only for the scope of the loop/process.
	MemoryModeTransient MemoryMode = "transient"
	// MemoryModePersist indicates database-backed persistence for long-term state.
	MemoryModePersist MemoryMode = "persist"
)

// ChatMessage represents a single unit of communication in a chat session.
// This is a core type used by MemoryStore implementations.
type ChatMessage struct {
	// Role is the speaker (e.g., "user", "model").
	Role string `json:"role"`
	// Content is the message text.
	Content string `json:"content"`
}

// MemoryStore defines the interface for persistent storage of chat history.
// Implementations can be anything from in-memory maps to Redis or SQL databases.
type MemoryStore interface {
	// Read retrieves the chat history for a given session.
	//
	// Parameters:
	//   - ctx: Context for cancellation.
	//   - sessionID: The unique identifier for the session.
	//
	// Returns:
	//   - A slice of ChatMessages, or an error if retrieval fails.
	Read(ctx context.Context, sessionID string) ([]ChatMessage, error)

	// Write saves the chat history for a given session.
	//
	// Parameters:
	//   - ctx: Context for cancellation.
	//   - sessionID: The unique identifier for the session.
	//   - msgs: The slice of ChatMessages to store.
	//
	// Returns:
	//   - An error if the write fails.
	Write(ctx context.Context, sessionID string, msgs []ChatMessage) error
}

// NoOpStore is a no-op implementation of MemoryStore for stateless mode.
// It discards writes and returns empty reads.
type NoOpStore struct{}

// Read returns nil, nil (empty history).
func (n NoOpStore) Read(_ context.Context, _ string) ([]ChatMessage, error) { return nil, nil }

// Write returns nil (successful no-op).
func (n NoOpStore) Write(_ context.Context, _ string, _ []ChatMessage) error { return nil }
