package supervisor

import (
	"context"
	"fmt"

	coreTypes "github.com/duynguyendang/manglekit-wip/core"
)

// legacyActionAdapter wraps core.Action into the v2 Action interface,
// forwarding Execute calls through the domain.Envelope types.
type legacyActionAdapter struct {
	inner       coreTypes.Action
	evaluator   coreTypes.Evaluator
	failureMode string
}

// NewSupervisedAction creates a supervised action for legacy SDK compatibility.
// It wraps a core.Action from the v1 SDK with basic policy evaluation.
func NewSupervisedAction(action coreTypes.Action, engine coreTypes.Evaluator, failureMode string) coreTypes.Action {
	return &legacyActionAdapter{
		inner:       action,
		evaluator:   engine,
		failureMode: failureMode,
	}
}

// NewSupervisedActionWithTracer creates a supervised action with tracing for legacy SDK compatibility.
func NewSupervisedActionWithTracer(action coreTypes.Action, engine coreTypes.Evaluator, tracer coreTypes.Tracer, failureMode string) coreTypes.Action {
	// Tracer is available for future integration; for now we just forward to the basic supervisor.
	return NewSupervisedAction(action, engine, failureMode)
}

func (a *legacyActionAdapter) Execute(ctx context.Context, input coreTypes.Envelope) (coreTypes.Envelope, error) {
	// Pre-flight: Evaluate policy against action metadata
	if a.evaluator != nil {
		meta := a.inner.Metadata()
		facts := []string{
			fmt.Sprintf("action_request(\"%s\", \"%s\").", meta.Name, meta.Type),
		}
		if err := a.evaluator.LoadFacts(facts); err != nil && a.failureMode == "closed" {
			return coreTypes.Envelope{}, fmt.Errorf("supervisor pre-flight policy check failed: %w", err)
		}
	}

	// Execute inner action
	result, err := a.inner.Execute(ctx, input)
	if err != nil {
		return coreTypes.Envelope{}, err
	}

	return result, nil
}

func (a *legacyActionAdapter) Metadata() coreTypes.ActionMetadata {
	return a.inner.Metadata()
}
