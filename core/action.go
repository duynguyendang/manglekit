package core

import "context"

// ActionMetadata provides metadata about an action.
// It is used for routing, observability, and debugging.
type ActionMetadata struct {
	// Name is the unique identifier for the action (e.g., "generate-content").
	Name string
	// Type describes the category of the action (e.g., "llm", "tool", "rag").
	Type string
	// InputContentType specifies the expected input format (Struct or JSON).
	InputContentType ContentType
	// InputType is the string name of the Go input type (e.g., "StockReq").
	InputType string
	// OutputType is the string name of the Go output type (e.g., "StockRes").
	OutputType string
	// IsDynamic indicates if the input type is generic (e.g., map[string]any or JSON).
	IsDynamic bool
}

// Action defines the interface for a unit of work in the Manglekit system.
// Any component that processes data (LLMs, databases, external APIs) must implement this interface.
type Action interface {
	// Execute performs the action's logic.
	//
	// It accepts a context for cancellation/timeout and an input Envelope containing the data.
	// It returns a new Envelope containing the result or an error if execution failed.
	Execute(ctx context.Context, input Envelope) (Envelope, error)

	// Metadata returns the action's metadata, including its name and type.
	Metadata() ActionMetadata
}
