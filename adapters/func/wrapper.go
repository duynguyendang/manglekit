package function

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/core"
)

// ToolFunc defines the signature for a generic Go function that can be wrapped as a tool.
// It accepts a context and a strongly-typed input, and returns a strongly-typed output and an error.
type ToolFunc[In any, Out any] func(context.Context, In) (Out, error)

// Wrapper adapts a generic ToolFunc into the universal core.Action interface.
// This allows any regular Go function to be managed, guarded, and traced by Manglekit.
type Wrapper[In any, Out any] struct {
	name        string
	fn          ToolFunc[In, Out]
	contentType core.ContentType
}

// New creates a new Action wrapper for the provided function.
//
// Parameters:
//   - name: The unique name of the action.
//   - fn: The function to wrap.
//
// Returns:
//   - A pointer to the initialized Wrapper.
func New[In any, Out any](name string, fn ToolFunc[In, Out]) *Wrapper[In, Out] {
	return &Wrapper[In, Out]{
		name:        name,
		fn:          fn,
		contentType: core.TypeStruct, // Default
	}
}

// SetContentType allows configuring the expected input content type (e.g., JSON vs Struct).
func (w *Wrapper[In, Out]) SetContentType(ct core.ContentType) {
	w.contentType = ct
}

// Execute performs the action logic.
// It automatically handles type assertion from the generic Envelope payload to the function's input type,
// and wraps the function's output back into an Envelope.
//
// Parameters:
//   - ctx: The execution context.
//   - input: The input Envelope.
//
// Returns:
//   - The result Envelope, or an error if type assertion or execution fails.
func (w *Wrapper[In, Out]) Execute(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	in, ok := input.Payload.(In)
	if !ok {
		return core.Envelope{}, fmt.Errorf("%w: invalid input type, expected %T but got %T", core.ErrSystemError, *new(In), input.Payload)
	}

	out, err := w.fn(ctx, in)
	if err != nil {
		return core.Envelope{}, err
	}

	return core.NewEnvelope(out), nil
}

// Metadata returns the action's metadata (name and type "function").
func (w *Wrapper[In, Out]) Metadata() core.ActionMetadata {
	return core.ActionMetadata{
		Name:             w.name,
		Type:             "function",
		InputContentType: w.contentType,
	}
}
