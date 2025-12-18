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

// Embedder abstracts the conversion of text to vector embeddings.
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
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

// HistoryStore defines the interface for persistent storage of chat history.
// Implementations can be anything from in-memory maps to Redis or SQL databases.
type HistoryStore interface {
	// Read retrieves the chat history for a given session.
	Read(ctx context.Context, sessionID string) ([]Message, error)

	// Append adds new messages to the history.
	Append(ctx context.Context, sessionID string, msgs []Message) error
}

// NopStore is a no-op implementation of HistoryStore for stateless mode.
// It discards writes and returns empty reads.
type NopStore struct{}

// Read returns nil, nil (empty history).
func (n NopStore) Read(_ context.Context, _ string) ([]Message, error) { return nil, nil }

// Append returns nil (successful no-op).
func (n NopStore) Append(_ context.Context, _ string, _ []Message) error { return nil }

// NopVectorStore is a no-op implementation of VectorStore.
type NopVectorStore struct{}

func (n NopVectorStore) Search(_ context.Context, _ string, _ []float32, _ int) ([]Document, error) {
	return nil, nil
}
func (n NopVectorStore) Upsert(_ context.Context, _ string, _ []Document) error { return nil }

// NopEmbedder is a no-op implementation of Embedder.
type NopEmbedder struct{}

func (n NopEmbedder) Embed(_ context.Context, _ string) ([]float32, error) {
	return nil, nil
}
