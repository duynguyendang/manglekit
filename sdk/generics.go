package sdk

import (
	"context"
	"fmt"

	function "github.com/duynguyendang/manglekit-wip/adapters/func"
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
	wrappedAction := function.New(name, handler)

	c.RegisterAction(name, c.Supervise(wrappedAction)) // Wrap with Blueprint Supervisor

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
