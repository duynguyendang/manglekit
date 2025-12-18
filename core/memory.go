package core

import "context"

// AgentMemory defines the capability to store and retrieve past experiences.
// It unifies sequential History (chat logs) and Semantic Memory (RAG).
type AgentMemory interface {
	// --- Sequential History ---
	// Read retrieves the chat history for a given session.
	Read(ctx context.Context, sessionID string) ([]Message, error)
	// Append adds new messages to the history.
	Append(ctx context.Context, sessionID string, msgs []Message) error

	// --- Semantic Memory (RAG) ---
	// Recall retrieves relevant context based on the current query.
	Recall(ctx context.Context, query string) (string, error)
	// Memorize stores a new interaction (Input/Output) for future recall.
	Memorize(ctx context.Context, query string, answer string) error

	// Init performs any necessary setup (e.g. connecting to DB).
	Init(ctx context.Context) error
}
