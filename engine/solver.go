package engine

import (
	"context"
	"fmt"
	"os"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/engine/resources"
	"github.com/google/mangle/ast"
	"github.com/google/mangle/parse"
)

// PolicyEngine is the core decision-making component of Manglekit.
// It orchestrates the loading of policies, maintaining the Datalog runtime,
// and executing authorization (Pre-Check) and validation (Post-Check) logic.
// It also integrates with observability (Tracing/Logging) to provide transparent governance.
type PolicyEngine struct {
	tracer  core.Tracer
	logger  core.Logger
	runtime *MangleRuntime
}

// New creates a new PolicyEngine with default no-op observability.
// This is suitable for basic usage where tracing and structured logging are not required.
//
// Returns:
//   - A pointer to a new PolicyEngine instance.
func New() *PolicyEngine {
	return &PolicyEngine{
		tracer:  &core.NopTracer{},
		logger:  core.NopLogger{},
		runtime: NewMangleRuntime(),
	}
}

// NewWithTracer creates a new PolicyEngine with tracing enabled.
//
// Deprecated: Use NewWithObservability instead.
//
// Parameters:
//   - tracer: The tracer implementation to use.
//
// Returns:
//   - A pointer to a new PolicyEngine instance.
func NewWithTracer(tracer core.Tracer) *PolicyEngine {
	if tracer == nil {
		tracer = &core.NopTracer{}
	}
	return &PolicyEngine{
		tracer:  tracer,
		logger:  core.NopLogger{},
		runtime: NewMangleRuntime(),
	}
}

// NewWithObservability creates a new PolicyEngine with both tracing and logging enabled.
// This is the recommended constructor for production use.
//
// Parameters:
//   - tracer: The tracer implementation (e.g., OpenTelemetry).
//   - logger: The logger implementation.
//
// Returns:
//   - A pointer to a new PolicyEngine instance.
func NewWithObservability(tracer core.Tracer, logger core.Logger) *PolicyEngine {
	if tracer == nil {
		tracer = &core.NopTracer{}
	}
	if logger == nil {
		logger = core.NopLogger{}
	}
	return &PolicyEngine{
		tracer:  tracer,
		logger:  logger,
		runtime: NewMangleRuntime(),
	}
}

// RecordLineage records a data lineage relationship between a child and a parent.
// Note: In the current architecture, lineage is primarily handled via context propagation
// and tracing spans rather than explicit in-memory storage.
//
// Parameters:
//   - ctx: The context.
//   - childID: The ID of the derived data.
//   - parentID: The ID of the source data.
func (e *PolicyEngine) RecordLineage(ctx context.Context, childID, parentID string) {
	if e.tracer != nil {
		// Lineage linking is handled via context propagation in GuardedAction.
		// If explicit linking span events are needed, they can be added here.
	}
}

// Logger returns the engine's configured Logger instance.
// This allows other components (like GuardedAction) to reuse the engine's logger.
//
// Returns:
//   - The configured Logger, or a NopLogger if none was set.
func (e *PolicyEngine) Logger() core.Logger {
	if e.logger == nil {
		return core.NopLogger{}
	}
	return e.logger
}

// LoadKnowledge loads static knowledge (facts) from a Turtle (.ttl) file.
// These facts are persisted in the runtime and available for all subsequent evaluations.
//
// Parameters:
//   - path: The file path to the .ttl file.
//
// Returns:
//   - An error if loading or parsing fails.
func (e *PolicyEngine) LoadKnowledge(path string) error {
	if path == "" {
		return nil
	}

	// Load facts from knowledge store
	facts, err := resources.LoadFromPath(path)
	if err != nil {
		return fmt.Errorf("failed to load knowledge from %s: %w", path, err)
	}

	// Add to runtime
	if err := e.runtime.LoadFacts(facts); err != nil {
		return fmt.Errorf("failed to load knowledge facts into runtime: %w", err)
	}

	if e.logger != nil {
		e.logger.Debug("knowledge loaded", "path", path, "facts_count", len(facts))
	}
	return nil
}

// LoadFromPath loads policy rules and facts from the specified file system path.
// It delegates to the underlying MangleRuntime to parse and compile the rules.
//
// Parameters:
//   - path: File path, directory, or glob pattern.
//
// Returns:
//   - An error if the policy files cannot be read or are invalid.
func (e *PolicyEngine) LoadFromPath(path string) error {
	if path == "" {
		return nil
	}

	// Verify the file exists and is readable (pre-flight check)
	_, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read policy file: %w", err)
	}

	// Load the rules into the Mangle runtime
	if err := e.runtime.Load(path); err != nil {
		return fmt.Errorf("failed to load policy rules: %w", err)
	}

	// Log successful load (optional debug message)
	if e.logger != nil {
		e.logger.Debug("policy loaded from file", "path", path)
	}

	return nil
}

// Authorize performs the Pre-Check phase of governance.
// It checks if the input is allowed to proceed based on the loaded policies.
// If the `deny(Req)` predicate is derived, it returns `core.ErrPolicyViolation`.
//
// It automatically starts a tracing span (`Datalog.PreCheck`) and logs attributes.
//
// Parameters:
//   - ctx: The execution context.
//   - actionMeta: Metadata about the action being authorized.
//   - input: The input envelope containing the payload and security labels.
//
// Returns:
//   - core.ErrPolicyViolation if blocked, or nil if allowed.
func (e *PolicyEngine) Authorize(ctx context.Context, actionMeta core.ActionMetadata, input core.Envelope) error {
	if e.tracer == nil {
		return e.authorizeInternal(ctx, actionMeta, input)
	}

	ctx, span := e.tracer.Start(ctx, core.SpanPreCheck)
	defer span.End()

	// Log security labels to span attributes for audit
	if len(input.SecurityLabels) > 0 {
		span.SetAttr(core.AttrLabels, input.SecurityLabels)
	}

	err := e.authorizeInternal(ctx, actionMeta, input)
	if err != nil {
		span.Error(err)
	} else {
		span.SetAttr(core.AttrOutcome, core.OutcomeAllowed)
	}
	return err
}

// authorizeInternal executes the core authorization logic.
func (e *PolicyEngine) authorizeInternal(ctx context.Context, actionMeta core.ActionMetadata, input core.Envelope) error {
	if e.runtime == nil || e.runtime.programInfo == nil {
		return nil // No runtime or program loaded, allow by default
	}

	// Convert the input payload to Mangle facts
	facts, err := toMangleFacts(core.EntityInput, input.Payload)
	if err != nil {
		if e.logger != nil {
			e.logger.Debug("failed to convert input to facts", "error", err)
		}
		// Return actual error to allow Fail-Open handling
		return fmt.Errorf("fact conversion error: %w", err)
	}

	// Generate facts for security labels using safe helper
	labelFacts, err := LabelsToFacts(core.EntityInput, input.SecurityLabels)
	if err != nil {
		if e.logger != nil {
			e.logger.Error("failed to generate label facts", "error", err)
		}
		// Return actual error to allow Fail-Open handling
		return fmt.Errorf("label conversion error: %w", err)
	}

	for _, factStr := range labelFacts {
		atom, err := parse.Atom(factStr)
		if err != nil {
			if e.logger != nil {
				e.logger.Error("failed to parse label fact", "fact", factStr, "error", err)
			}
			// Return actual error
			return fmt.Errorf("label parsing error: %w", err)
		}
		facts = append(facts, atom)
	}

	// Execute the deny(Req) query
	denied, err := e.runtime.ExecuteQuery(facts, fmt.Sprintf(`deny("%s")`, core.EntityInput))
	if err != nil {
		if e.logger != nil {
			e.logger.Debug("policy evaluation failed", "error", err)
		}
		// Return actual error to allow Fail-Open handling
		return fmt.Errorf("policy evaluation error: %w", err)
	}

	if denied {
		if e.logger != nil {
			e.logger.Debug("policy violation detected", "action", actionMeta.Name)
		}
		return core.ErrPolicyViolation
	}

	return nil
}

// Validate performs the Post-Check phase of governance.
// It checks if the output is allowed to be returned to the caller.
// If the `deny(Output)` predicate is derived, it returns `core.ErrPolicyViolation`.
//
// It automatically starts a tracing span (`Datalog.PostCheck`) and logs attributes.
//
// Parameters:
//   - ctx: The execution context.
//   - actionMeta: Metadata about the action being validated.
//   - output: The output envelope containing the result.
//
// Returns:
//   - The validated envelope (potentially modified, though currently pass-through), or an error.
func (e *PolicyEngine) Validate(ctx context.Context, actionMeta core.ActionMetadata, output core.Envelope) (core.Envelope, error) {
	if e.tracer == nil {
		return e.validateInternal(ctx, actionMeta, output)
	}

	ctx, span := e.tracer.Start(ctx, core.SpanPostCheck)
	defer span.End()

	// Log security labels to span attributes for audit
	if len(output.SecurityLabels) > 0 {
		span.SetAttr(core.AttrLabels, output.SecurityLabels)
	}

	result, err := e.validateInternal(ctx, actionMeta, output)
	if err != nil {
		span.Error(err)
		return core.Envelope{}, err
	}
	span.SetAttr(core.AttrOutcome, core.OutcomeAllowed)
	return result, nil
}

// validateInternal executes the core validation logic.
func (e *PolicyEngine) validateInternal(ctx context.Context, actionMeta core.ActionMetadata, output core.Envelope) (core.Envelope, error) {
	if e.runtime == nil || e.runtime.programInfo == nil {
		return output, nil // No runtime or program loaded, allow by default
	}

	// Convert the output payload to Mangle facts
	facts, err := toMangleFacts(core.EntityOutput, output.Payload)
	if err != nil {
		if e.logger != nil {
			e.logger.Debug("failed to convert output to facts", "error", err)
		}
		// Return actual error to allow Fail-Open handling
		return core.Envelope{}, fmt.Errorf("fact conversion error: %w", err)
	}

	// Generate facts for security labels using safe helper
	labelFacts, err := LabelsToFacts(core.EntityOutput, output.SecurityLabels)
	if err != nil {
		if e.logger != nil {
			e.logger.Error("failed to generate label facts", "error", err)
		}
		// Return actual error
		return core.Envelope{}, fmt.Errorf("label conversion error: %w", err)
	}

	for _, factStr := range labelFacts {
		atom, err := parse.Atom(factStr)
		if err != nil {
			if e.logger != nil {
				e.logger.Error("failed to parse label fact", "fact", factStr, "error", err)
			}
			// Return actual error
			return core.Envelope{}, fmt.Errorf("label parsing error: %w", err)
		}
		facts = append(facts, atom)
	}

	// Execute the deny(Output) query for post-check validation
	denied, err := e.runtime.ExecuteQuery(facts, fmt.Sprintf(`deny("%s")`, core.EntityOutput))
	if err != nil {
		if e.logger != nil {
			e.logger.Debug("post-check validation failed", "error", err)
		}
		// Return actual error to allow Fail-Open handling
		return core.Envelope{}, fmt.Errorf("policy evaluation error: %w", err)
	}

	if denied {
		if e.logger != nil {
			e.logger.Debug("post-check validation violation detected", "action", actionMeta.Name)
		}
		return core.Envelope{}, core.ErrPolicyViolation
	}

	return output, nil
}

// EvaluateSteering executes "Steering Policies" which determine what to do next.
// Unlike Authorize/Validate (which are binary Allow/Deny), Steering returns decisions like "Retry" or "Route".
//
// Logic Priority:
//  1. Correction (Retry): If `correction(Req, Hint)` is derived, we return `RETRY` with the hint.
//  2. Routing (Route): If `next_step(Req, Target)` is derived, we return `ROUTE` with the target.
//  3. Default: `ALLOW` (Proceed as normal).
//
// Parameters:
//   - ctx: The execution context.
//   - input: The input envelope.
//
// Returns:
//   - decision: The decision string (e.g., "RETRY", "ROUTE", "ALLOW").
//   - metadata: A map containing steering details (e.g., {"feedback": "hint"}).
//   - error: An error if evaluation fails.
func (e *PolicyEngine) EvaluateSteering(ctx context.Context, input core.Envelope) (string, map[string]string, error) {
	decision := core.DecisionAllow
	metadata := make(map[string]string)

	if e.runtime == nil || e.runtime.programInfo == nil {
		return decision, metadata, nil
	}

	// Convert the input payload to Mangle facts
	// We use "Req" as the entity ID, consistent with Authorize
	facts, err := toMangleFacts(core.EntityInput, input.Payload)
	if err != nil {
		if e.logger != nil {
			e.logger.Debug("failed to convert input to facts for steering", "error", err)
		}
		return decision, metadata, fmt.Errorf("failed to convert input to facts: %w", err)
	}

	// 1. Check Correction (Retry)
	// Query: correction("Req", Hint)
	_ = e.runtime.QueryWithSolutions(facts, fmt.Sprintf(`correction("%s", Hint)`, core.EntityInput), func(solution map[string]any) error {
		if hint, ok := solution["Hint"].(string); ok {
			decision = core.DecisionRetry
			metadata[core.KeyFeedback] = hint
			// Stop searching after first match
			return fmt.Errorf("found") // Use error to break early
		}
		return nil
	})

	if decision == core.DecisionRetry {
		return decision, metadata, nil
	}

	// 2. Check Routing
	// Query: next_step("Req", Target)
	_ = e.runtime.QueryWithSolutions(facts, fmt.Sprintf(`next_step("%s", Target)`, core.EntityInput), func(solution map[string]any) error {
		if target, ok := solution["Target"].(string); ok {
			decision = core.DecisionRoute
			metadata[core.KeyNextStep] = target
			return fmt.Errorf("found") // Use error to break early
		}
		return nil
	})

	return decision, metadata, nil
}

// toMangleFacts helper converts a Go struct to Mangle atoms via the Reflection API.
func toMangleFacts(entityID string, input any) ([]ast.Atom, error) {
	if input == nil {
		return nil, nil
	}

	var atoms []ast.Atom

	// Use the existing ToFacts helper to get string representations
	facts, err := ToFacts(entityID, input)
	if err != nil {
		return nil, err
	}

	// Parse each fact string back into an ast.Atom
	for _, factStr := range facts {
		atom, err := parse.Atom(factStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse fact '%s': %w", factStr, err)
		}
		atoms = append(atoms, atom)
	}

	return atoms, nil
}
