package supervisor

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/telemetry"

	"go.opentelemetry.io/otel"
)

// SupervisedAction is a decorator that wraps any `core.Action` to enforce governance blueprints.
// It implements the standard "Trace -> Assess -> Execute -> Reflect" lifecycle.
//
// Lifecycle:
//  1. Trace: Starts an OpenTelemetry span for the operation.
//  2. Assess: Checks Pre-Check blueprints (e.g., "infeasible(Req)").
//  3. Execute: Runs the inner action (e.g., calls the LLM).
//  4. Reflect: Checks Post-Check blueprints (e.g., "infeasible(Output)").
//  5. Steering: Evaluates steering blueprints for routing or correction.
type SupervisedAction struct {
	inner       core.Action
	engine      core.Evaluator
	tracer      core.Tracer
	failureMode string
}

// NewSupervisedAction creates a new SupervisedAction with default settings (no tracing).
//
// Parameters:
//   - action: The inner action to supervise.
//   - eng: The policy engine (evaluator) to use for governance.
//   - failureMode: The resilience strategy ("open" or "closed").
//
// Returns:
//   - A new SupervisedAction instance.
func NewSupervisedAction(action core.Action, eng core.Evaluator, failureMode string) *SupervisedAction {
	return &SupervisedAction{
		inner:       action,
		engine:      eng,
		tracer:      &core.NopTracer{},
		failureMode: failureMode,
	}
}

// NewSupervisedActionWithTracer creates a new SupervisedAction with tracing enabled.
//
// Parameters:
//   - action: The inner action to supervise.
//   - eng: The policy engine (evaluator).
//   - tracer: The tracer implementation.
//   - failureMode: "open" (log only on system error) or "closed" (block on system error).
//
// Returns:
//   - A new SupervisedAction instance.
func NewSupervisedActionWithTracer(action core.Action, eng core.Evaluator, tracer core.Tracer, failureMode string) *SupervisedAction {
	if tracer == nil {
		tracer = &core.NopTracer{}
	}
	return &SupervisedAction{
		inner:       action,
		engine:      eng,
		tracer:      tracer,
		failureMode: failureMode,
	}
}

// Execute runs the supervised action, orchestrating the full governance lifecycle.
//
// It performs the following steps:
//  1. Starts a span.
//  2. Injects the logger into the context.
//  3. Runs Assess(). If it fails, execution halts (unless Fail-Open).
//  4. Runs the inner Action.Execute().
//  5. Propagates taint labels from input to output.
//  6. Runs Reflect(). If it fails, the result is blocked.
//  7. Runs EvaluateSteering() to determine next steps (Retry/Route).
//
// Parameters:
//   - ctx: The execution context.
//   - input: The data envelope.
//
// Returns:
//   - The result envelope (possibly modified by blueprint), or an error.
func (g *SupervisedAction) Execute(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	// Auto-Tracing (Phase 5)
	// We use the injected core.Tracer if available, otherwise fallback to global OTel.
	tracer := g.tracer
	if tracer == nil {
		tracer = telemetry.NewOTelTracer(otel.Tracer("manglekit"))
	}
	meta := g.inner.Metadata()

	ctx, span := tracer.Start(ctx, fmt.Sprintf("Action.%s", meta.Name))
	defer span.End()

	span.SetAttributes(map[string]any{
		"mangle.action_name": meta.Name,
		"mangle.action_type": string(meta.Type),
		"mangle.input_id":    input.ID.String(),
	})

	result, err := g.executeInternal(ctx, input)
	if err != nil {
		span.RecordError(err)
		span.SetStatus("error", err.Error())
		// Distinguish between Blueprint HALT and System ERROR
		if g.isAlignmentIssue(err) {
			span.SetAttributes(map[string]any{core.AttrOutcome: core.OutcomeHalt})
			var alignErr *core.AlignmentError
			if errors.As(err, &alignErr) {
				attrs := map[string]any{core.KeyFeedback: alignErr.Message}
				if alignErr.RuleID != "" {
					attrs[core.AttrRuleID] = alignErr.RuleID
				}
				span.SetAttributes(attrs)
			} else {
				span.SetAttributes(map[string]any{core.KeyFeedback: err.Error()})
			}
		} else {
			span.SetAttributes(map[string]any{core.AttrOutcome: "ERROR"})
		}
		return core.Envelope{}, err
	}

	// Success Path: Determine outcome (Proceed/Route/Retry)
	decision := result.Metadata[core.KeyDecision]
	switch decision {
	case core.DecisionRetry:
		span.SetAttributes(map[string]any{core.AttrOutcome: "retry"})
		if hint, ok := result.Metadata[core.KeyFeedback]; ok {
			if s, ok := hint.(string); ok {
				span.SetAttributes(map[string]any{core.KeyFeedback: s})
			}
		}
	case core.DecisionRoute:
		span.SetAttributes(map[string]any{core.AttrOutcome: "route"})
		if target, ok := result.Metadata[core.KeyNextStep]; ok {
			if s, ok := target.(string); ok {
				span.SetAttributes(map[string]any{core.AttrActionName: s})
			}
		}
	default:
		span.SetAttributes(map[string]any{core.AttrOutcome: core.OutcomeProceed})
	}

	// Inject Retry Count if present
	if attemptVal, ok := input.Metadata["retry_count"]; ok {
		// handle both string and int
		if s, ok := attemptVal.(string); ok {
			if n, err := strconv.Atoi(s); err == nil {
				span.SetAttributes(map[string]any{core.AttrAttempt: n})
			}
		} else if n, ok := attemptVal.(int); ok {
			span.SetAttributes(map[string]any{core.AttrAttempt: n})
		}
	}

	// Set output ID without overwriting the outcome
	span.SetAttributes(map[string]any{
		"mangle.output_id": result.ID.String(),
	})
	return result, nil
}

// isAlignmentIssue checks if the error is a wrapped alignment check violation
func (g *SupervisedAction) isAlignmentIssue(err error) bool {
	return core.IsAlignmentError(err)
}

// Metadata delegates to the inner action's Metadata method.
// This allows the SupervisedAction to transparently represent the underlying capability.
func (g *SupervisedAction) Metadata() core.ActionMetadata {
	return g.inner.Metadata()
}

// shouldBlock determines if the action should be blocked based on the error and failure mode.
func (g *SupervisedAction) shouldBlock(err error) bool {
	if err == nil {
		return false
	}
	// Always block on explicit alignment issues
	if core.IsAlignmentError(err) {
		return true
	}
	// Always block on Input/Validation errors (bypass fail-open)
	if core.IsInputError(err) {
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
// It orchestrates: Logger Setup → Pre-Check → Config → Execute → Post-Check → Steering.
func (g *SupervisedAction) executeInternal(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	// Inject logger and get metadata
	ctx = core.ContextWithLogger(ctx, g.engine.Logger())
	logger := core.LoggerFromContext(ctx)
	meta := g.inner.Metadata()

	logger.Info("Action started", "action", meta.Name, "input_id", input.ID.String())

	// Phase 1: Pre-Check (Assessment)
	if err := g.performAssessment(ctx, logger, meta, input); err != nil {
		return core.Envelope{}, err
	}

	// Phase 2: Dynamic Configuration Injection
	g.injectDynamicConfig(ctx, logger, &input)

	// Phase 3: Execute Inner Action
	result, err := g.executeAction(ctx, logger, meta, input)
	if err != nil {
		return core.Envelope{}, err
	}

	// Phase 4: Post-Check (Reflection)
	validatedResult, err := g.performReflection(ctx, logger, meta, result)
	if err != nil {
		return core.Envelope{}, err
	}

	// Phase 5: Steering
	validatedResult = g.applySteering(ctx, logger, meta, validatedResult)

	logger.Info("Action completed", "action", meta.Name, "result", "success")
	return validatedResult, nil
}

// performAssessment executes the pre-check phase (blueprint validation of input).
func (g *SupervisedAction) performAssessment(ctx context.Context, logger core.Logger, meta core.ActionMetadata, input core.Envelope) error {
	if err := g.engine.Assess(ctx, meta, input); err != nil {
		if g.shouldBlock(err) {
			msg := "assessment failed"
			if core.IsInputError(err) {
				msg = "assessment blocked due to invalid input"
			}
			logger.Warn(msg, core.AttrActionName, meta.Name, "error", err.Error())
			return fmt.Errorf("%s: %w", msg, err)
		}
		// Fail-Open
		logger.Warn("engine assessment failed but Fail-Open active. Proceeding.", "error", err)
	}
	return nil
}

// injectDynamicConfig queries the engine for configuration overrides and injects them into input metadata.
func (g *SupervisedAction) injectDynamicConfig(ctx context.Context, logger core.Logger, input *core.Envelope) {
	config, err := g.engine.GetActionConfig(ctx, *input)
	if err != nil {
		logger.Warn("failed to retrieve action config", "error", err)
		return
	}

	if len(config) == 0 {
		return
	}

	if input.Metadata == nil {
		input.Metadata = make(map[string]any)
	}
	for k, v := range config {
		input.Metadata[core.PrefixPromptConfig+k] = v
	}
}

// executeAction runs the inner action with context propagation and security label inheritance.
func (g *SupervisedAction) executeAction(ctx context.Context, logger core.Logger, meta core.ActionMetadata, input core.Envelope) (core.Envelope, error) {
	// Context Propagation: set input ID as parent
	childCtx := core.WithParentID(ctx, input.ID.String())

	// Execute
	result, err := g.inner.Execute(childCtx, input)
	if err != nil {
		logger.Error("action execution failed", core.AttrActionName, meta.Name, "error", err.Error())
		return core.Envelope{}, fmt.Errorf("action execution failed: %w", err)
	}

	// Security label propagation
	if len(input.SecurityLabels) > 0 {
		result.MergeLabels(input.SecurityLabels)
	}

	// Link output to input
	if result.Metadata == nil {
		result.Metadata = make(map[string]any)
	}
	result.Metadata["derived_from"] = input.ID.String()

	return result, nil
}

// performReflection executes the post-check phase (blueprint validation of output).
func (g *SupervisedAction) performReflection(ctx context.Context, logger core.Logger, meta core.ActionMetadata, result core.Envelope) (core.Envelope, error) {
	validatedResult, err := g.engine.Reflect(ctx, meta, result)
	if err != nil {
		if g.shouldBlock(err) {
			msg := "reflection failed"
			if core.IsInputError(err) {
				msg = "reflection blocked due to invalid input"
			}
			logger.Warn(msg, "action", meta.Name, "error", err.Error())
			return core.Envelope{}, fmt.Errorf("%s: %w", msg, err)
		}
		// Fail-Open: use original result
		logger.Warn("engine reflection failed but Fail-Open active. Proceeding.", "error", err)
		validatedResult = result
	}
	return validatedResult, nil
}

// applySteering evaluates steering decisions and stamps metadata.
func (g *SupervisedAction) applySteering(ctx context.Context, logger core.Logger, meta core.ActionMetadata, result core.Envelope) core.Envelope {
	decision, steeringMeta, err := g.engine.EvaluateSteering(ctx, result)
	if err != nil {
		logger.Warn("steering evaluation failed", "action", meta.Name, "error", err.Error())
		// Continue with empty decision on steering error
		decision = ""
		steeringMeta = nil
	}

	// Stamp metadata
	if result.Metadata == nil {
		result.Metadata = make(map[string]any)
	}
	result.Metadata[core.KeyDecision] = decision
	for k, v := range steeringMeta {
		result.Metadata[k] = v
	}

	return result
}
