package guard

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/engine"
)

// GuardedAction wraps a core.Action to enforce policies.
// It provides the main transaction span for tracing the entire execution flow.
type GuardedAction struct {
	inner  core.Action
	engine *engine.PolicyEngine
	tracer core.Tracer
}

// New creates a new GuardedAction without tracing (for backward compatibility).
func New(action core.Action, eng *engine.PolicyEngine) *GuardedAction {
	return &GuardedAction{
		inner:  action,
		engine: eng,
		tracer: &core.NopTracer{},
	}
}

// NewWithTracer creates a new GuardedAction with tracing enabled.
func NewWithTracer(action core.Action, eng *engine.PolicyEngine, tracer core.Tracer) *GuardedAction {
	if tracer == nil {
		tracer = &core.NopTracer{}
	}
	return &GuardedAction{
		inner:  action,
		engine: eng,
		tracer: tracer,
	}
}

// Execute runs the action through the policy engine's checks.
// The entire execution flow (Authorize → Inner Action → Validate) is wrapped
// in a top-level span for full observability.
func (g *GuardedAction) Execute(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	// If no tracer is configured, execute without tracing
	if g.tracer == nil {
		return g.executeInternal(ctx, input)
	}

	// Get action metadata for span naming
	meta := g.inner.Metadata()

	// Start the main transaction span named after the action
	ctx, span := g.tracer.Start(ctx, fmt.Sprintf("Action.%s", meta.Name))
	defer span.End()

	// Set attributes on the span
	span.SetAttr("action.name", meta.Name)
	span.SetAttr("action.type", meta.Type)

	result, err := g.executeInternal(ctx, input)
	if err != nil {
		span.Error(err)
		return core.Envelope{}, err
	}

	span.SetAttr("outcome", "success")
	return result, nil
}

// executeInternal contains the actual execution logic.
// It receives the context with the active span so child spans can be created.
// The logger is injected into the context here, ensuring all downstream code
// can access it via core.LoggerFromContext(ctx).
func (g *GuardedAction) executeInternal(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	// Inject the logger into the context for downstream access
	ctx = core.LoggerWithContext(ctx, g.engine.Logger())

	logger := core.LoggerFromContext(ctx)
	meta := g.inner.Metadata()
	logger.Debug("starting action execution",
		"action.name", meta.Name,
		"action.type", meta.Type,
		"envelope.id", input.ID.String(),
	)

	// 1. Ingestion: Link Input to Parent
	if parentID, ok := core.GetParentID(ctx); ok {
		g.engine.RecordLineage(ctx, input.ID.String(), parentID)
	}

	// 2. Pre-Check: Authorization
	if err := g.engine.Authorize(ctx, g.inner.Metadata(), input); err != nil {
		logger.Warn("authorization failed",
			"action.name", meta.Name,
			"error", err.Error(),
		)
		return core.Envelope{}, fmt.Errorf("authorization failed: %w", err)
	}

	// 3. Propagation: Pass the Gene
	// Propagate the current input ID as the new parent for the inner action
	childCtx := core.WithParentID(ctx, input.ID.String())

	// 4. Execution: Run inner action
	result, err := g.inner.Execute(childCtx, input)
	if err != nil {
		logger.Error("action execution failed",
			"action.name", meta.Name,
			"error", err.Error(),
		)
		return core.Envelope{}, fmt.Errorf("action execution failed: %w", err)
	}

	// 5. Linking: Link Output to Input
	g.engine.RecordLineage(ctx, result.ID.String(), input.ID.String())
	if result.Metadata == nil {
		result.Metadata = make(map[string]string)
	}
	result.Metadata["derived_from"] = input.ID.String()

	// 6. Post-Check: Validation
	validatedResult, err := g.engine.Validate(ctx, g.inner.Metadata(), result)
	if err != nil {
		logger.Warn("validation failed",
			"action.name", meta.Name,
			"error", err.Error(),
		)
		return core.Envelope{}, fmt.Errorf("validation failed: %w", err)
	}

	logger.Debug("action execution completed successfully",
		"action.name", meta.Name,
		"action.type", meta.Type,
	)

	return validatedResult, nil
}

// Metadata returns the metadata of the inner action.
func (g *GuardedAction) Metadata() core.ActionMetadata {
	return g.inner.Metadata()
}
