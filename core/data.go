package core

import (
	"context"
)

// --- Knowledge (RAG) ---

// VectorStore abstracts Semantic Search.
type VectorStore interface {
	Search(ctx context.Context, collection string, vector []float32, k int) ([]Document, error)
	Upsert(ctx context.Context, collection string, docs []Document) error
}

// FactLoader loads external data (Graph/RDF) into the Engine.
type FactLoader interface {
	LoadFacts(ctx context.Context, source string) ([]string, error)
}

// --- Memory (State) ---

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

// MemoryStore defines the interface for persistent storage of chat history.
// Implementations can be anything from in-memory maps to Redis or SQL databases.
type MemoryStore interface {
	// Read retrieves the chat history for a given session.
	Read(ctx context.Context, sessionID string) ([]Message, error)

	// Write saves the chat history for a given session.
	Write(ctx context.Context, sessionID string, msgs []Message) error
}

// NopStore is a no-op implementation of MemoryStore for stateless mode.
// It discards writes and returns empty reads.
type NopStore struct{}

// Read returns nil, nil (empty history).
func (n NopStore) Read(_ context.Context, _ string) ([]Message, error) { return nil, nil }

// Write returns nil (successful no-op).
func (n NopStore) Write(_ context.Context, _ string, _ []Message) error { return nil }
