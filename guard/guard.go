package guard

import (
	"context"
	"errors"
	"fmt"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/engine"
)

// GuardedAction wraps a core.Action to enforce policies.
// It provides the main transaction span for tracing the entire execution flow.
type GuardedAction struct {
	inner       core.Action
	engine      *engine.PolicyEngine
	tracer      core.Tracer
	failureMode string
}

// New creates a new GuardedAction without tracing (for backward compatibility).
func New(action core.Action, eng *engine.PolicyEngine, failureMode string) *GuardedAction {
	return &GuardedAction{
		inner:       action,
		engine:      eng,
		tracer:      &core.NopTracer{},
		failureMode: failureMode,
	}
}

// NewWithTracer creates a new GuardedAction with tracing enabled.
func NewWithTracer(action core.Action, eng *engine.PolicyEngine, tracer core.Tracer, failureMode string) *GuardedAction {
	if tracer == nil {
		tracer = &core.NopTracer{}
	}
	return &GuardedAction{
		inner:       action,
		engine:      eng,
		tracer:      tracer,
		failureMode: failureMode,
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
	span.SetAttr(core.AttrActionName, meta.Name)
	span.SetAttr(core.AttrActionType, meta.Type)

	result, err := g.executeInternal(ctx, input)
	if err != nil {
		span.Error(err)
		return core.Envelope{}, err
	}

	span.SetAttr(core.AttrOutcome, core.OutcomeSuccess)
	return result, nil
}

// shouldBlock determines if the action should be blocked based on the error and failure mode.
func (g *GuardedAction) shouldBlock(err error) bool {
	if err == nil {
		return false
	}
	// Always block on explicit policy violations
	if errors.Is(err, core.ErrPolicyViolation) {
		return true
	}
	// If mode is "open" (Fail-Open), allow execution (return false)
	// Otherwise (default/closed), block execution (return true)
	if g.failureMode == "open" {
		return false
	}
	return true
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
		core.AttrActionName, meta.Name,
		core.AttrActionType, meta.Type,
		"envelope.id", input.ID.String(),
	)

	// 1. Ingestion: Link Input to Parent (Tracing only)
	if parentID, ok := core.GetParentID(ctx); ok {
		g.engine.RecordLineage(ctx, input.ID.String(), parentID)
	}

	// 2. Pre-Check: Authorization
	if err := g.engine.Authorize(ctx, g.inner.Metadata(), input); err != nil {
		if g.shouldBlock(err) {
			logger.Warn("authorization failed",
				core.AttrActionName, meta.Name,
				"error", err.Error(),
			)
			return core.Envelope{}, fmt.Errorf("authorization failed: %w", err)
		}
		// Fail-Open
		logger.Warn("engine failed but Fail-Open active. Proceeding.", "error", err)
	}

	// 3. Context Propagation: Pass the Gene
	// Propagate the current input ID as the new parent for the inner action
	childCtx := core.WithParentID(ctx, input.ID.String())

	// 4. Execution: Run inner action
	result, err := g.inner.Execute(childCtx, input)
	if err != nil {
		logger.Error("action execution failed",
			core.AttrActionName, meta.Name,
			"error", err.Error(),
		)
		return core.Envelope{}, fmt.Errorf("action execution failed: %w", err)
	}

	// 5. Propagation: Output inherits Input's security labels
	if len(input.SecurityLabels) > 0 {
		result.MergeLabels(input.SecurityLabels)
	}

	// 6. Linking: Link Output to Input (Tracing only)
	g.engine.RecordLineage(ctx, result.ID.String(), input.ID.String())
	if result.Metadata == nil {
		result.Metadata = make(map[string]string)
	}
	result.Metadata["derived_from"] = input.ID.String()

	// 7. Post-Check: Validation
	validatedResult, err := g.engine.Validate(ctx, g.inner.Metadata(), result)
	if err != nil {
		if g.shouldBlock(err) {
			logger.Warn("validation failed",
				core.AttrActionName, meta.Name,
				"error", err.Error(),
			)
			return core.Envelope{}, fmt.Errorf("validation failed: %w", err)
		}
		// Fail-Open: use result as validatedResult
		logger.Warn("engine validation failed but Fail-Open active. Proceeding.", "error", err)
		validatedResult = result
	}

	// 8. Steering: Evaluate next steps (Correction/Routing)
	decision, steeringMeta, err := g.engine.EvaluateSteering(ctx, validatedResult)
	if err != nil {
		logger.Warn("steering evaluation failed",
			core.AttrActionName, meta.Name,
			"error", err.Error(),
		)
		return core.Envelope{}, fmt.Errorf("steering evaluation failed: %w", err)
	}

	// Stamp metadata
	if validatedResult.Metadata == nil {
		validatedResult.Metadata = make(map[string]string)
	}
	validatedResult.Metadata[core.KeyDecision] = decision
	for k, v := range steeringMeta {
		validatedResult.Metadata[k] = v
	}

	logger.Debug("action execution completed successfully",
		core.AttrActionName, meta.Name,
		core.AttrActionType, meta.Type,
		"decision", decision,
	)

	return validatedResult, nil
}

// Metadata returns the metadata of the inner action.
func (g *GuardedAction) Metadata() core.ActionMetadata {
	return g.inner.Metadata()
}
