package guard

import (
	"context"
	"errors"
	"fmt"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/engine"
)

// GuardedAction is a decorator that wraps any `core.Action` to enforce governance policies.
// It implements the standard "Trace -> Authorize -> Execute -> Validate" lifecycle.
//
// Lifecycle:
//  1. Trace: Starts an OpenTelemetry span for the operation.
//  2. Authorize: Checks Pre-Check policies (e.g., "deny(Req)").
//  3. Execute: Runs the inner action (e.g., calls the LLM).
//  4. Validate: Checks Post-Check policies (e.g., "deny(Output)").
//  5. Steering: Evaluates steering policies for routing or correction.
type GuardedAction struct {
	inner       core.Action
	engine      *engine.PolicyEngine
	tracer      core.Tracer
	failureMode string
}

// New creates a new GuardedAction with default settings (no tracing).
//
// Parameters:
//   - action: The inner action to protect.
//   - eng: The policy engine to use for governance.
//   - failureMode: The resilience strategy ("open" or "closed").
//
// Returns:
//   - A new GuardedAction instance.
func New(action core.Action, eng *engine.PolicyEngine, failureMode string) *GuardedAction {
	return &GuardedAction{
		inner:       action,
		engine:      eng,
		tracer:      &core.NopTracer{},
		failureMode: failureMode,
	}
}

// NewWithTracer creates a new GuardedAction with tracing enabled.
//
// Parameters:
//   - action: The inner action to protect.
//   - eng: The policy engine.
//   - tracer: The tracer implementation.
//   - failureMode: "open" (log only on system error) or "closed" (block on system error).
//
// Returns:
//   - A new GuardedAction instance.
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

// Execute runs the guarded action, orchestrating the full governance lifecycle.
//
// It performs the following steps:
//  1. Starts a span.
//  2. Injects the logger into the context.
//  3. Runs Authorize(). If it fails, execution halts (unless Fail-Open).
//  4. Runs the inner Action.Execute().
//  5. Propagates taint labels from input to output.
//  6. Runs Validate(). If it fails, the result is blocked.
//  7. Runs EvaluateSteering() to determine next steps (Retry/Route).
//
// Parameters:
//   - ctx: The execution context.
//   - input: The data envelope.
//
// Returns:
//   - The result envelope (possibly modified by policy), or an error.
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

// Metadata delegates to the inner action's Metadata method.
// This allows the GuardedAction to transparently represent the underlying capability.
func (g *GuardedAction) Metadata() core.ActionMetadata {
	return g.inner.Metadata()
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
	ctx = core.ContextWithLogger(ctx, g.engine.Logger())

	logger := core.LoggerFromContext(ctx)
	meta := g.inner.Metadata()
	logger.Info("Action started",
		"action", meta.Name,
		"input_id", input.ID.String(),
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
				"action", meta.Name,
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
			"action", meta.Name,
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

	logger.Info("Action completed",
		"action", meta.Name,
		"result", "success", // Simplified as per doc
	)

	return validatedResult, nil
}
