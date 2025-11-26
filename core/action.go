package core

import "context"

// ActionMetadata provides metadata about an action.
type ActionMetadata struct {
	Name string
	Type string // e.g., "llm", "tool", "rag"
}

// Action defines the interface for a unit of work.
type Action interface {
	Execute(ctx context.Context, input Envelope) (Envelope, error)
	Metadata() ActionMetadata
}
