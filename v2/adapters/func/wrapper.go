package function

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/v2/core"
)

// ToolFunc is a generic function signature for tools.
type ToolFunc[In any, Out any] func(context.Context, In) (Out, error)

// Wrapper is a generic struct that wraps a ToolFunc to implement the core.Action interface.
type Wrapper[In any, Out any] struct {
	name string
	fn   ToolFunc[In, Out]
}

// New creates a new Wrapper for the given function.
func New[In any, Out any](name string, fn ToolFunc[In, Out]) *Wrapper[In, Out] {
	return &Wrapper[In, Out]{
		name: name,
		fn:   fn,
	}
}

// Execute unpacks the input, calls the wrapped function, and packs the output.
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

// Metadata returns the metadata for the action.
func (w *Wrapper[In, Out]) Metadata() core.ActionMetadata {
	return core.ActionMetadata{
		Name: w.name,
		Type: "function",
	}
}
