package sdk

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/core"
)

// Execute processes an envelope by determining the initial action via the policy engine.
// It relies on the 'next_step' predicate to route the request.
func (c *Client) Execute(ctx context.Context, input core.Envelope, opts ...ExecuteOption) (core.Envelope, error) {
	// 1. Evaluate Steering to find the entry point
	decision, meta, err := c.engine.EvaluateSteering(ctx, input)
	if err != nil {
		return core.Envelope{}, fmt.Errorf("failed to evaluate entry route: %w", err)
	}

	if decision == core.DecisionRoute {
		actionName := meta[core.KeyNextStep]
		if actionName == "" {
			return core.Envelope{}, fmt.Errorf("routing decision returned empty action name")
		}
		// Forward payload and metadata
		return c.ExecuteByName(ctx, actionName, input.Payload, opts...)
	}

	return core.Envelope{}, fmt.Errorf("no execution route defined for this input (decision=%s, meta=%v)", decision, meta)
}
