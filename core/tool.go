package core

import (
	"context"
)

// ExecutionContext holds the state and data for a single declarative run.
type ExecutionContext struct {
	// Input is the initial query text from the user.
	Input string
	// Query is the structured query, which may be mutated by tools.
	Query Query
	// Documents are the documents retrieved from a retriever.
	Documents []Doc
	// Answer is the final answer to be returned to the user.
	Answer Answer
	// A map for arbitrary metadata that can be passed between tools.
	Meta map[string]any
}

// Tool is the interface for a single, executable step in a declarative pipeline.
type Tool interface {
	// Execute performs the tool's action, modifying the context.
	Execute(ctx context.Context, execCtx *ExecutionContext) error
}
