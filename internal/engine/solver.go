package engine

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/engine/resources"
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
	pe := &PolicyEngine{
		tracer:  &core.NopTracer{},
		logger:  core.NopLogger{},
		runtime: NewMangleRuntime(),
	}

	// Auto-load Standard Library
	if err := pe.runtime.AddPolicy(resources.GetStdLib()); err != nil {
		panic("manglekit: failed to load std.dl: " + err.Error())
	}

	return pe
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

	pe := &PolicyEngine{
		tracer:  tracer,
		logger:  logger,
		runtime: NewMangleRuntime(),
	}

	// Load Planner Core Rules
	if err := pe.runtime.AddPolicy(resources.GetPlannerRules()); err != nil {
		if logger != nil {
			logger.Error("failed to load planner core schema", "error", err)
		}
	}

	// Load Manglekit Standard Library (std.dl)
	if err := pe.runtime.AddPolicy(resources.GetStdLib()); err != nil {
		if logger != nil {
			logger.Error("failed to load standard library", "error", err)
		}
		// Failure to load stdlib is critical for dynamic features
		panic("manglekit: failed to load std.dl: " + err.Error())
	}

	return pe
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

// LoadFacts injects a list of raw Datalog fact strings into the runtime's base knowledge.
// This allows adding dynamic context or configuration at runtime (e.g., feature flags).
//
// Parameters:
//   - facts: A slice of Datalog fact strings.
//
// Returns:
//   - An error if parsing or evaluation fails.
func (e *PolicyEngine) LoadFacts(facts []string) error {
	return e.runtime.LoadFacts(facts)
}

// RegisterActionMetadata injects metadata about a registered action into the Datalog runtime.
// It generates facts that describe the action's interface, enabling the Planner to reason about it.
//
// Generated Facts:
//   - action("name")
//   - has_input("name", "InputType")
//   - has_output("name", "OutputType")
//
// Parameters:
//   - meta: The metadata of the action.
//
// Returns:
//   - An error if fact loading fails.
func (e *PolicyEngine) RegisterActionMetadata(meta core.ActionMetadata) error {
	var facts []string
	safeName := escapeString(meta.Name)

	facts = append(facts, fmt.Sprintf("action(\"%s\")", safeName))

	if meta.InputType != "" {
		facts = append(facts, fmt.Sprintf("has_input(\"%s\", \"%s\")", safeName, escapeString(meta.InputType)))
	}

	if meta.OutputType != "" {
		facts = append(facts, fmt.Sprintf("has_output(\"%s\", \"%s\")", safeName, escapeString(meta.OutputType)))
	}

	return e.LoadFacts(facts)
}

// LoadPolicy loads policy rules from a raw Datalog string.
// This decouples the engine from file I/O.
//
// Parameters:
//   - policy: The Datalog rules as a string.
//
// Returns:
//   - An error if parsing or loading fails.
func (e *PolicyEngine) LoadPolicy(policy string) error {
	if policy == "" {
		return nil
	}

	// Load the rules into the Mangle runtime
	if err := e.runtime.AddPolicy(policy); err != nil {
		return fmt.Errorf("failed to load policy: %w", err)
	}

	// Log successful load
	if e.logger != nil {
		e.logger.Debug("policy loaded from string", "length", len(policy))
	}

	return nil
}

// Authorize performs the Pre-Check phase of governance.
// It checks if the input is allowed to proceed based on the loaded policies.
// If the `deny(Req)` predicate is derived, it returns `core.ErrAlignment`.
//
// It automatically starts a tracing span (`Datalog.PreCheck`) and logs attributes.
//
// Parameters:
//   - ctx: The execution context.
//   - actionMeta: Metadata about the action being authorized.
//   - input: The input envelope containing the payload and security labels.
//
// Returns:
//   - core.ErrAlignment if blocked, or nil if allowed.
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
	facts, err := toMangleFacts(core.EntityInput, input.Payload, input.ContentType)
	if err != nil {
		if e.logger != nil {
			e.logger.Debug("failed to convert input to facts", "error", err)
		}
		// Return actual error to allow Fail-Open handling
		return fmt.Errorf("fact conversion error: %w", err)
	}

	// Inject Action Metadata facts
	// action_operation("Req", "Name")
	if actionMeta.Name != "" {
		safeName := core.EntityInput
		safeOp := actionMeta.Name
		opFactStr := fmt.Sprintf("action_operation(\"%s\", \"%s\")", escapeString(safeName), escapeString(safeOp))
		opAtom, err := parse.Atom(opFactStr)
		if err == nil {
			facts = append(facts, opAtom)
		}
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
			e.logger.Debug("alignment issue detected", "action", actionMeta.Name)
		}
		return core.ErrAlignment
	}

	return nil
}

// Validate performs the Post-Check phase of governance.
// It checks if the output is allowed to be returned to the caller.
// If the `deny(Output)` predicate is derived, it returns `core.ErrAlignment`.
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
	facts, err := toMangleFacts(core.EntityOutput, output.Payload, output.ContentType)
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
		// Teacher-Student Protocol: Try to extract a human-readable violation message
		// Query: violation_msg(Msg)
		var violationMsg string
		// We reuse the facts which already contain the Output payload
		_ = e.runtime.QueryWithSolutions(facts, "violation_msg(Msg)", func(solution map[string]any) error {
			if msg, ok := solution["Msg"].(string); ok {
				violationMsg = msg
				return fmt.Errorf("found") // Stop searching
			}
			return nil
		})

		if e.logger != nil {
			e.logger.Debug("post-check validation violation detected", "action", actionMeta.Name, "violation_msg", violationMsg)
		}

		if violationMsg != "" {
			return core.Envelope{}, &core.AlignmentError{Message: violationMsg}
		}
		return core.Envelope{}, core.ErrAlignment
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
	facts, err := toMangleFacts(core.EntityInput, input.Payload, input.ContentType)
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

// ExecuteQuery executes a raw Datalog query against the current program state.
// It wraps the underlying runtime execution with observability (tracing).
//
// Parameters:
//   - ctx: The execution context.
//   - facts: Temporary facts to include in this specific query execution.
//   - queryStr: The Datalog query atom.
//
// Returns:
//   - true if derived, false otherwise.
//   - error if execution fails.
func (e *PolicyEngine) ExecuteQuery(ctx context.Context, facts []ast.Atom, queryStr string) (bool, error) {
	if e.tracer == nil {
		return e.runtime.ExecuteQuery(facts, queryStr)
	}

	ctx, span := e.tracer.Start(ctx, "Datalog.ExecuteQuery")
	defer span.End()

	span.SetAttr("datalog.query", queryStr)

	res, err := e.runtime.ExecuteQuery(facts, queryStr)
	if err != nil {
		span.Error(err)
		return false, err
	}

	span.SetAttr("datalog.result", res)
	return res, nil
}

// Query executes a Datalog query and returns all matching solutions.
// Each solution is a map where keys are variable names (e.g., "Action") and values are stringified constants.
//
// Parameters:
//   - ctx: The execution context.
//   - facts: Temporary facts (strings) to include.
//   - queryStr: The Datalog query with variables (e.g., 'plan_step(Action, Order)').
//
// Returns:
//   - A list of solution maps.
//   - An error if execution fails.
func (e *PolicyEngine) Query(ctx context.Context, facts []string, queryStr string) ([]map[string]string, error) {
	var results []map[string]string

	if e.runtime == nil {
		return nil, fmt.Errorf("runtime not initialized")
	}

	// Parse temporary facts
	var atomFacts []ast.Atom
	for _, f := range facts {
		atom, err := parse.Atom(f)
		if err != nil {
			return nil, fmt.Errorf("failed to parse fact '%s': %w", f, err)
		}
		atomFacts = append(atomFacts, atom)
	}

	if e.tracer != nil {
		var span core.Span
		ctx, span = e.tracer.Start(ctx, "Datalog.Query")
		defer span.End()
		span.SetAttr("datalog.query", queryStr)
	}

	err := e.runtime.QueryWithSolutions(atomFacts, queryStr, func(solution map[string]any) error {
		// Convert map[string]any to map[string]string
		strMap := make(map[string]string)
		for k, v := range solution {
			if s, ok := v.(string); ok {
				strMap[k] = s
			} else {
				strMap[k] = fmt.Sprintf("%v", v)
			}
		}
		results = append(results, strMap)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return results, nil
}

// toMangleFacts helper converts a Go struct to Mangle atoms via the Reflection API.
func toMangleFacts(entityID string, input any, contentType core.ContentType) ([]ast.Atom, error) {
	if input == nil {
		return nil, nil
	}

	var atoms []ast.Atom
	var facts []string
	var err error

	// Choose strategy based on ContentType
	if contentType == core.TypeJSON {
		facts, err = Flatten(entityID, input)
	} else {
		// Default to Reflection (Struct)
		facts, err = ToFacts(entityID, input)
	}

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
