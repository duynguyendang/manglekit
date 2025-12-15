package core

import "context"

// AgentMemory defines the capability to store and retrieve past experiences.
// This interface supports the RAG (Retrieval-Augmented Generation) pattern,
// allowing the agent to recall relevant context from a semantic store.
type AgentMemory interface {
	// Recall retrieves relevant context based on the current query.
	// Returns a single string (consolidated context) to be injected into the prompt.
	Recall(ctx context.Context, query string) (string, error)

	// Memorize stores a new interaction (Input/Output) for future recall.
	Memorize(ctx context.Context, query string, answer string) error

	// Init performs any necessary setup (e.g. connecting to DB).
	Init(ctx context.Context) error
}
