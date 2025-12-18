package core

import "context"

// AgentMemory defines the capability to store and retrieve past experiences.
// It abstracts underlying Vector DBs (e.g., Qdrant, Pinecone).
type AgentMemory interface {
	// Recall retrieves relevant context based on the current query.
	Recall(ctx context.Context, query string) (string, error)

	// Memorize stores a new interaction (Input/Output) for future recall.
	Memorize(ctx context.Context, query string, answer string) error

	// Init performs any necessary setup (e.g. connecting to DB).
	Init(ctx context.Context) error
}
