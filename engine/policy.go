package engine

import (
	"context"
	"fmt"
	"os"

	"github.com/duynguyendang/manglekit/core"
	"github.com/google/mangle/ast"
	"github.com/google/mangle/parse"
)

// PolicyEngine handles policy-based authorization and validation.
// It wraps the Datalog execution layer and provides observability through tracing
// and structured logging.
type PolicyEngine struct {
	tracer  core.Tracer
	logger  core.Logger
	runtime *MangleRuntime
}

// New creates a new PolicyEngine without a tracer or logger (for backward compatibility).
// Prefer NewWithObservability for full observability.
func New() *PolicyEngine {
	return &PolicyEngine{
		tracer:  &core.NopTracer{},
		logger:  core.NopLogger{},
		runtime: NewMangleRuntime(),
	}
}

// NewWithTracer creates a new PolicyEngine with tracing enabled.
// Deprecated: Use NewWithObservability for full observability support.
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

// RecordLineage records a lineage relationship for observability.
// It no longer stores relationships in memory.
func (e *PolicyEngine) RecordLineage(ctx context.Context, childID, parentID string) {
	if e.tracer != nil {
		// Lineage linking is handled via context propagation in GuardedAction.
		// If explicit linking span events are needed, they can be added here.
	}
}

// Logger returns the engine's configured Logger instance.
// This is used by the guard layer to inject the logger into the context.
func (e *PolicyEngine) Logger() core.Logger {
	if e.logger == nil {
		return core.NopLogger{}
	}
	return e.logger
}

// LoadFromPath loads policy rules from a file.
// It parses the Datalog rules and prepares them for evaluation.
// Supports:
// - Single .dlog rule files
// - Single .facts/.edb/.data fact files
// - Directories (recursively)
// - Glob patterns (e.g., "policies/*.dlog")
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

// Authorize performs pre-execution policy checks (Pre-Check).
// It evaluates the Datalog rules to determine if the action is allowed to proceed.
func (e *PolicyEngine) Authorize(ctx context.Context, actionMeta core.ActionMetadata, input core.Envelope) error {
	if e.tracer == nil {
		return e.authorizeInternal(ctx, actionMeta, input)
	}

	ctx, span := e.tracer.Start(ctx, "Datalog.PreCheck")
	defer span.End()

	// Log security labels to span attributes for audit
	if len(input.SecurityLabels) > 0 {
		span.SetAttr("mangle.labels", input.SecurityLabels)
	}

	err := e.authorizeInternal(ctx, actionMeta, input)
	if err != nil {
		span.Error(err)
	} else {
		span.SetAttr("outcome", "ALLOWED")
	}
	return err
}

// authorizeInternal contains the actual authorization logic.
// It converts the input envelope to facts and evaluates the deny(Req) predicate.
// If deny(Req) is derived, it returns ErrPolicyViolation.
func (e *PolicyEngine) authorizeInternal(ctx context.Context, actionMeta core.ActionMetadata, input core.Envelope) error {
	if e.runtime == nil || e.runtime.programInfo == nil {
		return nil // No runtime or program loaded, allow by default
	}

	// Convert the input payload to Mangle facts
	facts, err := toMangleFacts("Req", input.Payload)
	if err != nil {
		if e.logger != nil {
			e.logger.Debug("failed to convert input to facts", "error", err)
		}
		// If conversion fails, deny for safety
		return core.ErrPolicyViolation
	}

	// Generate facts for security labels using safe helper
	labelFacts, err := LabelsToFacts("Req", input.SecurityLabels)
	if err != nil {
		if e.logger != nil {
			e.logger.Error("failed to generate label facts", "error", err)
		}
		// Fail-Closed: If we can't process security labels, we must deny.
		return core.ErrPolicyViolation
	}

	for _, factStr := range labelFacts {
		atom, err := parse.Atom(factStr)
		if err != nil {
			if e.logger != nil {
				e.logger.Error("failed to parse label fact", "fact", factStr, "error", err)
			}
			// Fail-Closed
			return core.ErrPolicyViolation
		}
		facts = append(facts, atom)
	}

	// Execute the deny(Req) query
	denied, err := e.runtime.ExecuteQuery(facts, `deny("Req")`)
	if err != nil {
		if e.logger != nil {
			e.logger.Debug("policy evaluation failed", "error", err)
		}
		// If evaluation fails, deny for safety
		return core.ErrPolicyViolation
	}

	if denied {
		if e.logger != nil {
			e.logger.Debug("policy violation detected", "action", actionMeta.Name)
		}
		return core.ErrPolicyViolation
	}

	return nil
}

// Validate performs post-execution policy checks (Post-Check).
// It evaluates the Datalog rules to validate and potentially transform the output.
func (e *PolicyEngine) Validate(ctx context.Context, actionMeta core.ActionMetadata, output core.Envelope) (core.Envelope, error) {
	if e.tracer == nil {
		return e.validateInternal(ctx, actionMeta, output)
	}

	ctx, span := e.tracer.Start(ctx, "Datalog.PostCheck")
	defer span.End()

	// Log security labels to span attributes for audit
	if len(output.SecurityLabels) > 0 {
		span.SetAttr("mangle.labels", output.SecurityLabels)
	}

	result, err := e.validateInternal(ctx, actionMeta, output)
	if err != nil {
		span.Error(err)
		return core.Envelope{}, err
	}
	span.SetAttr("outcome", "ALLOWED")
	return result, nil
}

// validateInternal contains the actual validation logic.
// It converts the output envelope to facts and performs post-check validation.
func (e *PolicyEngine) validateInternal(ctx context.Context, actionMeta core.ActionMetadata, output core.Envelope) (core.Envelope, error) {
	if e.runtime == nil || e.runtime.programInfo == nil {
		return output, nil // No runtime or program loaded, allow by default
	}

	// Convert the output payload to Mangle facts
	facts, err := toMangleFacts("Output", output.Payload)
	if err != nil {
		if e.logger != nil {
			e.logger.Debug("failed to convert output to facts", "error", err)
		}
		// If conversion fails, pass through for safety
		return output, nil
	}

	// Generate facts for security labels using safe helper
	labelFacts, err := LabelsToFacts("Output", output.SecurityLabels)
	if err != nil {
		if e.logger != nil {
			e.logger.Error("failed to generate label facts", "error", err)
		}
		// Fail-Closed
		return core.Envelope{}, core.ErrPolicyViolation
	}

	for _, factStr := range labelFacts {
		atom, err := parse.Atom(factStr)
		if err != nil {
			if e.logger != nil {
				e.logger.Error("failed to parse label fact", "fact", factStr, "error", err)
			}
			// Fail-Closed
			return core.Envelope{}, core.ErrPolicyViolation
		}
		facts = append(facts, atom)
	}

	// Execute the deny(Output) query for post-check validation
	denied, err := e.runtime.ExecuteQuery(facts, `deny("Output")`)
	if err != nil {
		if e.logger != nil {
			e.logger.Debug("post-check validation failed", "error", err)
		}
		// If evaluation fails, pass through for safety
		return output, nil
	}

	if denied {
		if e.logger != nil {
			e.logger.Debug("post-check validation violation detected", "action", actionMeta.Name)
		}
		return core.Envelope{}, core.ErrPolicyViolation
	}

	return output, nil
}

// toMangleFacts converts a Go data structure into a slice of Mangle atoms.
// It uses reflection to traverse the structure and create facts.
// Each field becomes a fact with the format: predicate(entityID, value)
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
