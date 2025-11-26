package guard

import (
	"context"

	"github.com/duynguyendang/manglekit/v2/core"
	"github.com/duynguyendang/manglekit/v2/engine"
)

// GuardedAction wraps a core.Action to enforce policies.
type GuardedAction struct {
	inner  core.Action
	engine *engine.PolicyEngine
}

// New creates a new GuardedAction.
func New(action core.Action, eng *engine.PolicyEngine) *GuardedAction {
	return &GuardedAction{
		inner:  action,
		engine: eng,
	}
}

// Execute runs the action through the policy engine's checks.
func (g *GuardedAction) Execute(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	if err := g.engine.Authorize(ctx, g.inner.Metadata(), input); err != nil {
		return core.Envelope{}, err
	}

	result, err := g.inner.Execute(ctx, input)
	if err != nil {
		return core.Envelope{}, err
	}

	validatedResult, err := g.engine.Validate(ctx, g.inner.Metadata(), result)
	if err != nil {
		return core.Envelope{}, err
	}

	return validatedResult, nil
}

// Metadata returns the metadata of the inner action.
func (g *GuardedAction) Metadata() core.ActionMetadata {
	return g.inner.Metadata()
}
