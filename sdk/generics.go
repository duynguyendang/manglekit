package sdk

import (
	"context"
	"fmt"
	"github.com/duynguyendang/manglekit/core"
    "github.com/duynguyendang/manglekit/internal/util"
)

// Runnable is a handle for a type-safe action
type Runnable[In any, Out any] struct {
	client *Client
	name   string
}

// Define registers a pure Go function as an Action
func Define[In any, Out any](
	c *Client,
	name string,
	handler func(context.Context, In) (Out, error),
) *Runnable[In, Out] {

	// Adapter: Convert Typed Handler -> Core Action
	wrappedAction := &util.FuncWrapper{
		ActionName: name,
		Fn: func(ctx context.Context, env core.Envelope) (core.Envelope, error) {
			var input In
            // Type Assertion on Payload
			if p, ok := env.Payload.(In); ok {
				input = p
			} else {
                // TODO: Add JSON map decoding logic here if needed for dynamic inputs
				return core.Envelope{}, fmt.Errorf("input type mismatch: expected %T, got %T", input, env.Payload)
			}

			outData, err := handler(ctx, input)
			if err != nil {
				return core.Envelope{}, err
			}

			return core.NewEnvelope(outData), nil
		},
	}

	c.RegisterAction(name, c.Protect(wrappedAction)) // Wrap with Policy Guard

	return &Runnable[In, Out]{
		client: c,
		name:   name,
	}
}

// Run executes the action with strict types
func (r *Runnable[In, Out]) Run(ctx context.Context, input In) (Out, error) {
	// ExecuteByName will wrap the input in an envelope
	resEnv, err := r.client.ExecuteByName(ctx, r.name, input)

    var zero Out
	if err != nil {
		return zero, err
	}

	if out, ok := resEnv.Payload.(Out); ok {
		return out, nil
	}
	return zero, fmt.Errorf("output type mismatch: expected %T, got %T", zero, resEnv.Payload)
}
