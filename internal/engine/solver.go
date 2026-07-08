package engine

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"codeberg.org/TauCeti/mangle-go/ast"
	"codeberg.org/TauCeti/mangle-go/parse"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/engine/resources"
)

var ErrSolutionFound = errors.New("solution found")

// PolicyEngine is the core decision-making component of Manglekit.
// It orchestrates the loading of policies, maintaining the Datalog runtime,
// and executing authorization (Pre-Check) and validation (Post-Check) logic.
// It also integrates with observability (Tracing/Logging) to provide transparent governance.
type PolicyEngine struct {
	tracer       core.Tracer
	logger       core.Logger
	runtime      *MangleRuntime
	queryTimeout time.Duration
}

// New creates a new PolicyEngine with default no-op observability.
// This is suitable for basic usage where tracing and structured logging are not required.
//
// Returns:
//   - A pointer to a new PolicyEngine instance.
func New() (*PolicyEngine, error) {
	pe := &PolicyEngine{
		tracer:  &core.NopTracer{},
		logger:  core.NopLogger{},
		runtime: NewMangleRuntime(),
	}

	// Auto-load Standard Library
	if err := pe.runtime.AddPolicy(context.Background(), resources.StdLib()); err != nil {
		return nil, fmt.Errorf("manglekit: failed to load std.dl: %w", err)
	}

	return pe, nil
}

// RegisterExternalPredicate adds an external predicate to the policy engine.
// External predicates allow Datalog rules to call Go functions for operations
// like HTTP requests, database lookups, time checks, etc.
//
// Example usage:
//
//	engine.RegisterExternalPredicate("http_get", func(ctx context.Context, inputs []any) ([][]any, error) {
//	    url := inputs[0].(string)
//	    resp, err := http.Get(url)
//	    if err != nil {
//	        return nil, err
//	    }
//	    return [][]any{{resp.StatusCode}}, nil
//	})
//
// Then use in policy:
//
//	http_allowed(URL) :- http_get(URL, Status), Status >= 200, Status < 300.
//
// Note: External predicates are evaluated at runtime and can introduce
// non-determinism. Use with caution in security-critical policies.
func (e *PolicyEngine) RegisterExternalPredicate(name string, fn func(ctx context.Context, inputs []any) ([][]any, error)) error {
	return e.runtime.RegisterExternalPredicate(name, fn)
}

// ExternalPredicateRegistry returns the external predicate registry for inspection.
func (e *PolicyEngine) ExternalPredicateRegistry() *ExternalPredicateRegistry {
	return e.runtime.ExternalPredicates()
}

// Runtime returns the underlying MangleRuntime for advanced usage.
// Use with caution - directly manipulating the runtime bypasses PolicyEngine safeguards.
func (e *PolicyEngine) Runtime() *MangleRuntime {
	return e.runtime
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
func NewWithObservability(tracer core.Tracer, logger core.Logger) (*PolicyEngine, error) {
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
	if err := pe.runtime.AddPolicy(context.Background(), resources.GetPlannerRules()); err != nil {
		if logger != nil {
			logger.Error("failed to load planner core schema", "error", err)
		}
	}

	// Load Manglekit Standard Library (std.dl)
	if err := pe.runtime.AddPolicy(context.Background(), resources.StdLib()); err != nil {
		if logger != nil {
			logger.Error("failed to load standard library", "error", err)
		}
		// Failure to load stdlib is critical for dynamic features
		return nil, fmt.Errorf("manglekit: failed to load std.dl: %w", err)
	}

	return pe, nil
}

// WithQueryTimeout sets the maximum duration for a single query execution.
// If a query exceeds this timeout, it returns context.DeadlineExceeded.
// A zero or negative duration means no timeout (unlimited).
func (e *PolicyEngine) WithQueryTimeout(timeout time.Duration) *PolicyEngine {
	e.queryTimeout = timeout
	return e
}

// RecordLineage records a data lineage relationship between a child and a parent.
// It emits a span event if tracing is enabled.
func (e *PolicyEngine) RecordLineage(ctx context.Context, childID, parentID string) {
	if e.tracer == nil {
		return
	}

	_, span := e.tracer.Start(ctx, "Lineage.Record")
	defer span.End()

	span.SetAttributes(map[string]any{
		"lineage.child":  childID,
		"lineage.parent": parentID,
	})
}

// Logger returns the engine's configured Logger instance.
// This allows other components (like SupervisedAction) to reuse the engine's logger.
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
func (e *PolicyEngine) LoadFacts(ctx context.Context, facts []string) error {
	return e.runtime.LoadFacts(ctx, facts)
}

// RegisterAction injects metadata about a registered action into the Datalog runtime.
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
func (e *PolicyEngine) RegisterAction(meta core.ActionMetadata) error {
	var facts []string
	safeName := escapeString(meta.Name)

	facts = append(facts, fmt.Sprintf("action(\"%s\")", safeName))

	if meta.InputType != "" {
		facts = append(facts, fmt.Sprintf("has_input(\"%s\", \"%s\")", safeName, escapeString(meta.InputType)))
	}

	if meta.OutputType != "" {
		facts = append(facts, fmt.Sprintf("has_output(\"%s\", \"%s\")", safeName, escapeString(meta.OutputType)))
	}

	return e.LoadFacts(context.Background(), facts)
}

// LoadPolicy loads policy rules from a raw Datalog string.
// This decouples the engine from file I/O.
//
// Parameters:
//   - ctx: The execution context (unused in current implementation but required by interface).
//   - policy: The Datalog rules as a string.
//
// Returns:
//   - An error if parsing or loading fails.
func (e *PolicyEngine) LoadPolicy(ctx context.Context, policy string) error {
	if policy == "" {
		return nil
	}

	// Load the rules into the Mangle runtime
	if err := e.runtime.AddPolicy(ctx, policy); err != nil {
		return fmt.Errorf("failed to load policy: %w", err)
	}

	// Log successful load
	if e.logger != nil {
		e.logger.Debug("policy loaded from string", "length", len(policy))
	}

	return nil
}

// LoadFromSource loads a Datalog program from a raw string, replacing
// any existing program state. Unlike LoadPolicy (which routes to
// AddPolicy), this path also scans the external-predicate registry
// and auto-emits the matching `Decl ... external()` declarations.
// Use this when the policy references external predicates that were
// registered via RegisterExternalPredicate.
func (e *PolicyEngine) LoadFromSource(ctx context.Context, source string) error {
	_ = ctx
	if source == "" {
		return nil
	}
	if err := e.runtime.LoadFromSource(ctx, source); err != nil {
		return fmt.Errorf("failed to load policy from source: %w", err)
	}
	return nil
}

// LoadGherkinPolicy loads a Gherkin feature file and compiles it to Datalog.
// This enables BDD-style policy definitions using natural language.
//
// Parameters:
//   - ctx: The execution context.
//   - featureContent: The Gherkin feature file content.
//
// Returns:
//   - An error if parsing, compilation, or loading fails.
func (e *PolicyEngine) LoadGherkinPolicy(ctx context.Context, featureContent string) error {
	if featureContent == "" {
		return nil
	}

	// Compile Gherkin to Datalog
	compiler := NewGherkinCompiler()
	datalog, err := compiler.CompileFromString(featureContent)
	if err != nil {
		return fmt.Errorf("failed to compile Gherkin policy: %w", err)
	}

	// Log the generated Datalog if logger is available
	if e.logger != nil {
		e.logger.Debug("compiled Gherkin to Datalog", "datalog", datalog)
	}

	// Load the compiled Datalog into the runtime
	return e.LoadPolicy(ctx, datalog)
}

// AssessPlan implements the core.Evaluator interface.
// It performs a high-level assessment of the input, mapping Assess logic to a Decision.
// The Decision.AuditTrail is populated from the gate evaluation,
// carrying matched rules, tiers, latencies, and fact counts.
func (e *PolicyEngine) AssessPlan(ctx context.Context, input core.Envelope) (core.Decision, error) {
	trail, err := e.assessInternal(ctx, core.ActionMetadata{}, input)

	decision := core.Decision{
		AuditTrail: trail,
	}

	if err != nil {
		// If authorization fails, it's a DENY
		var alignErr *core.AlignmentError
		if errors.As(err, &alignErr) {
			decision.Outcome = core.DecisionHalt
			decision.Reasons = []string{alignErr.Message}
			decision.Meta = map[string]string{"rule_id": alignErr.RuleID}
			return decision, nil
		}
		decision.Outcome = core.DecisionHalt
		decision.Reasons = []string{err.Error()}
		return decision, err
	}

	decision.Outcome = core.DecisionProceed
	return decision, nil
}

// GetActionConfig queries the engine for dynamic configuration parameters.
// It executes the query `action_config(Key, Value)` and returns a map of results.
//
// Parameters:
//   - ctx: The execution context.
//   - input: The input envelope.
//
// Returns:
//   - A map of configuration keys and values.
//   - An error if execution fails.
func (e *PolicyEngine) GetActionConfig(ctx context.Context, input core.Envelope) (map[string]string, error) {
	config := make(map[string]string)

	if e.runtime == nil || e.runtime.programInfo == nil {
		return config, nil
	}

	// Convert input to facts
	facts, err := toMangleFacts(core.EntityInput, input.Payload, input.ContentType)
	if err != nil {
		if e.logger != nil {
			e.logger.Debug("failed to convert input to facts for config", "error", err)
		}
		// Return empty config on fact conversion failure to avoid blocking
		return config, nil
	}

	// Inject Envelope Facts
	for _, factStr := range input.Facts {
		atom, err := parse.Atom(factStr)
		if err != nil {
			if e.logger != nil {
				e.logger.Error("failed to parse envelop fact", "fact", factStr, "error", err)
			}
			// Continue without this fact
			continue
		}
		facts = append(facts, atom)
	}

	// Inject Metadata facts: meta(Key, Value) and attempt(N)
	for k, v := range input.Metadata {
		safeK := escapeString(k)

		// Handle slice values: meta("key", "val1"), meta("key", "val2")
		rv := reflect.ValueOf(v)
		if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
			for i := 0; i < rv.Len(); i++ {
				item := rv.Index(i).Interface()
				itemStr := fmt.Sprintf("%v", item)
				metaFact := fmt.Sprintf("meta(\"%s\", \"%s\")", safeK, escapeString(itemStr))
				if atom, err := parse.Atom(metaFact); err == nil {
					facts = append(facts, atom)
				}
			}
		} else {
			// Single value
			vStr := fmt.Sprintf("%v", v)
			safeV := escapeString(vStr)

			metaFact := fmt.Sprintf("meta(\"%s\", \"%s\")", safeK, safeV)
			if atom, err := parse.Atom(metaFact); err == nil {
				facts = append(facts, atom)
			}

			// attempt(N) from retry_count
			if k == "retry_count" {
				attemptFact := fmt.Sprintf("attempt(%s)", vStr)
				if atom, err := parse.Atom(attemptFact); err == nil {
					facts = append(facts, atom)
				}
			}
		}
	}

	// Execute query: config(Key, Value)
	err = e.runtime.QueryWithSolutions(ctx, facts, "config(Key, Value)", func(solution map[string]any) error {
		key, kOk := solution["Key"].(string)
		val, vOk := solution["Value"].(string)
		if kOk && vOk {
			config[key] = val
		}
		return nil
	})

	if err != nil && e.logger != nil {
		e.logger.Debug("failed to query action config", "error", err)
	}

	return config, nil
}

// ExtractActionArguments queries the engine for action arguments from Datalog bindings.
// This enables the PolicyEngine to dynamically extract arguments from Datalog rules
// and populate the ActionEnvelope.Arguments.
//
// Datalog pattern: suggested_action(ActionName, [key1: value1, key2: value2])
// Or: action_args(ActionName, Key, Value)
//
// Parameters:
//   - ctx: The execution context.
//   - input: The input envelope.
//   - actionName: The action name to extract arguments for.
//
// Returns:
//   - map[string]interface{} of extracted arguments.
//   - error if extraction fails.
func (e *PolicyEngine) ExtractActionArguments(ctx context.Context, input core.Envelope, actionName string) (map[string]interface{}, error) {
	args := make(map[string]interface{})

	if e.runtime == nil {
		return args, nil
	}

	// Convert input to facts
	facts, err := toMangleFacts(core.EntityInput, input.Payload, input.ContentType)
	if err != nil {
		return args, nil // Return empty on error
	}

	// Inject Envelope Facts
	for _, factStr := range input.Facts {
		atom, err := parse.Atom(factStr)
		if err != nil {
			continue
		}
		facts = append(facts, atom)
	}

	// Try to extract from action_args(ActionName, Key, Value) pattern
	queryStr := fmt.Sprintf("action_args(\"%s\", Key, Value)", actionName)
	err = e.runtime.QueryWithSolutions(ctx, nil, queryStr, func(solution map[string]any) error {
		if key, ok := solution["Key"]; ok {
			if keyStr, ok := key.(string); ok {
				if value, ok := solution["Value"]; ok {
					args[keyStr] = convertDatalogValue(value)
				}
			}
		}
		return nil
	})
	if err != nil {
		// Query failed, try alternative pattern
		e.logger.Debug("action_args query failed, trying alternative pattern", "error", err)
	}

	// Try alternative: suggested_action(ActionName, KeyValueList) pattern
	// Where KeyValueList is expected to be in format: [key1:val1, key2:val2]
	if len(args) == 0 {
		altQueryStr := fmt.Sprintf("suggested_action(\"%s\", Args)", actionName)
		err = e.runtime.QueryWithSolutions(ctx, nil, altQueryStr, func(solution map[string]any) error {
			if argsVal, ok := solution["Args"]; ok {
				if argsMap, ok := argsVal.(map[any]any); ok {
					for k, v := range argsMap {
						if keyStr, ok := k.(string); ok {
							args[keyStr] = convertDatalogValue(v)
						}
					}
				}
			}
			return nil
		})
		if err != nil {
			e.logger.Debug("suggested_action query failed", "error", err)
		}
	}

	return args, nil
}

// convertDatalogValue converts Datalog values to native Go types.
func convertDatalogValue(val any) interface{} {
	switch v := val.(type) {
	case string:
		return v
	case int, int8, int16, int32, int64:
		return v
	case uint, uint8, uint16, uint32, uint64:
		return v
	case float32, float64:
		return v
	case bool:
		return v
	case []any:
		// Convert list to slice
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = convertDatalogValue(item)
		}
		return result
	case map[any]any:
		// Convert map to map[string]interface{}
		result := make(map[string]interface{})
		for k, v := range v {
			if keyStr, ok := k.(string); ok {
				result[keyStr] = convertDatalogValue(v)
			}
		}
		return result
	default:
		return fmt.Sprintf("%v", v)
	}
}

// Assess performs the Pre-Check phase of governance.
// It checks if the input is allowed to proceed based on the loaded policies.
// If the `infeasible(Req, Reason)` or `deny(Req)` predicate is derived, it returns `core.ErrAlignment`.
//
// It automatically starts a tracing span (`Datalog.Assess`) and logs attributes.
//
// Parameters:
//   - ctx: The execution context.
//   - actionMeta: Metadata about the action being authorized.
//   - input: The input envelope containing the payload and security labels.
//
// Returns:
//   - core.ErrAlignment if blocked, or nil if allowed.
//
// Formerly: Authorize
func (e *PolicyEngine) Assess(ctx context.Context, actionMeta core.ActionMetadata, input core.Envelope) error {
	if e.tracer == nil {
		_, err := e.assessInternal(ctx, actionMeta, input)
		return err
	}

	ctx, span := e.tracer.Start(ctx, core.SpanPreCheck)
	defer span.End()

	// Log security labels to span attributes for audit
	if len(input.SecurityLabels) > 0 {
		span.SetAttributes(map[string]any{core.AttrLabels: input.SecurityLabels})
	}

	_, err := e.assessInternal(ctx, actionMeta, input)
	if err != nil {
		span.RecordError(err)
	} else {
		span.SetAttributes(map[string]any{core.AttrOutcome: core.OutcomeProceed})
	}
	return err
}

// assessInternal executes the core authorization logic.
// It returns both the error and the audit trail from the gate evaluation.
func (e *PolicyEngine) assessInternal(ctx context.Context, actionMeta core.ActionMetadata, input core.Envelope) (*core.AuditTrail, error) {
	var extraFacts []ast.Atom

	// Inject Action Metadata facts: action_operation("Req", "Name")
	if actionMeta.Name != "" {
		safeName := core.EntityInput
		safeOp := actionMeta.Name
		opFactStr := fmt.Sprintf("action_operation(\"%s\", \"%s\")", escapeString(safeName), escapeString(safeOp))
		opAtom, err := parse.Atom(opFactStr)
		if err == nil {
			extraFacts = append(extraFacts, opAtom)
		}
	}

	// Use infeasible(Req, Reason) with fallback to deny(Req)
	return e.evaluateGateWithTrail(ctx, actionMeta.Name, core.EntityInput, input, extraFacts...)
}

// Reflect performs the Post-Check phase of governance.
// It checks if the output is allowed to be returned to the caller.
// If the `infeasible(Output, Reason)` predicate is derived, it returns `core.ErrAlignment`.
//
// It automatically starts a tracing span (`Datalog.Reflect`) and logs attributes.
//
// Parameters:
//   - ctx: The execution context.
//   - actionMeta: Metadata about the action being validated.
//   - output: The output envelope containing the result.
//
// Returns:
//   - The validated envelope (potentially modified, though currently pass-through), or an error.
//
// Formerly: Validate
func (e *PolicyEngine) Reflect(ctx context.Context, actionMeta core.ActionMetadata, output core.Envelope) (core.Envelope, error) {
	if e.tracer == nil {
		return e.reflectInternal(ctx, actionMeta, output)
	}

	ctx, span := e.tracer.Start(ctx, core.SpanPostCheck)
	defer span.End()

	// Log security labels to span attributes for audit
	if len(output.SecurityLabels) > 0 {
		span.SetAttributes(map[string]any{core.AttrLabels: output.SecurityLabels})
	}

	result, err := e.reflectInternal(ctx, actionMeta, output)
	if err != nil {
		span.RecordError(err)
		return core.Envelope{}, err
	}
	span.SetAttributes(map[string]any{core.AttrOutcome: core.OutcomeProceed})
	return result, nil
}

// reflectInternal executes the core validation logic.
func (e *PolicyEngine) reflectInternal(ctx context.Context, actionMeta core.ActionMetadata, output core.Envelope) (core.Envelope, error) {
	err := e.evaluateGate(ctx, actionMeta.Name, core.EntityOutput, output)
	if err != nil {
		return core.Envelope{}, err
	}
	return output, nil
}

// tierPriority returns the priority of a tier (lower = more authoritative).
// A T0 axiom always outranks a T3 user rule regardless of predicate order.
func tierPriority(tier core.Tier) int {
	switch tier {
	case core.TierT0_Axiom:
		return 0
	case core.TierT1_Governance:
		return 1
	case core.TierT2_Playbook:
		return 2
	case core.TierT3_User:
		return 3
	default:
		return 4
	}
}

// evaluateGate centralizes the logic for "Check -> Deny -> Explain".
// It is used by both Assess (Pre-Check) and Reflect (Post-Check).
// This is a convenience wrapper that discards the audit trail.
func (e *PolicyEngine) evaluateGate(ctx context.Context, actionName string, entityID string, env core.Envelope, extraFacts ...ast.Atom) error {
	_, err := e.evaluateGateWithTrail(ctx, actionName, entityID, env, extraFacts...)
	return err
}

// evaluateGateWithTrail is like evaluateGate but also returns the audit trail
// explaining which rules matched and at what tier. The trail is populated from
// the actual halt/retry/route matches collected during evaluation.
func (e *PolicyEngine) evaluateGateWithTrail(ctx context.Context, actionName string, entityID string, env core.Envelope, extraFacts ...ast.Atom) (*core.AuditTrail, error) {
	if e.runtime == nil || e.runtime.programInfo == nil {
		return nil, &core.AlignmentError{Message: "policy engine not initialized: fail-closed"}
	}

	// 1. ToFacts: Convert Payload
	facts, err := toMangleFacts(entityID, env.Payload, env.ContentType)
	if err != nil {
		if e.logger != nil {
			e.logger.Debug("failed to convert payload to facts", "error", err)
		}
		// Wrap in InputError to ensure it is BLOCKED upstream
		return nil, &core.InputError{Err: fmt.Errorf("fact conversion error: %w", err)}
	}

	// 2. Inject Extra Facts (e.g. Action Operation)
	facts = append(facts, extraFacts...)

	// 3. Inject Labels
	labelFacts, err := LabelsToFacts(entityID, env.SecurityLabels)
	if err != nil {
		if e.logger != nil {
			e.logger.Error("failed to generate label facts", "error", err)
		}
		return nil, &core.InputError{Err: fmt.Errorf("label conversion error: %w", err)}
	}
	for _, f := range labelFacts {
		atom, err := parse.Atom(f)
		if err != nil {
			if e.logger != nil {
				e.logger.Error("failed to parse label fact", "fact", f, "error", err)
			}
			return nil, &core.InputError{Err: fmt.Errorf("label parsing error: %w", err)}
		}
		facts = append(facts, atom)
	}

	// 4. Inject Explicit Facts from Envelope
	for _, f := range env.Facts {
		atom, err := parse.Atom(f)
		if err != nil {
			if e.logger != nil {
				e.logger.Error("failed to parse envelop fact", "fact", f, "error", err)
			}
			return nil, &core.InputError{Err: fmt.Errorf("envelope fact parsing error: %w", err)}
		}
		facts = append(facts, atom)
	}

	// 5. Inject Metadata
	for k, v := range env.Metadata {
		safeK := escapeString(k)

		// Handle slice values: meta("key", "val1"), meta("key", "val2")
		rv := reflect.ValueOf(v)
		if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
			for i := 0; i < rv.Len(); i++ {
				item := rv.Index(i).Interface()
				itemStr := fmt.Sprintf("%v", item)
				metaFact := fmt.Sprintf("meta(\"%s\", \"%s\")", safeK, escapeString(itemStr))
				if atom, err := parse.Atom(metaFact); err == nil {
					facts = append(facts, atom)
				}
			}
		} else {
			// Single value
			vStr := fmt.Sprintf("%v", v)
			safeV := escapeString(vStr)

			metaFact := fmt.Sprintf("meta(\"%s\", \"%s\")", safeK, safeV)
			if atom, err := parse.Atom(metaFact); err == nil {
				facts = append(facts, atom)
			}

			// attempt(N) from retry_count
			if k == "retry_count" {
				attemptFact := fmt.Sprintf("attempt(%s)", vStr)
				if atom, err := parse.Atom(attemptFact); err == nil {
					facts = append(facts, atom)
				}
			}
		}
	}

	// 6. Run Query — Tier-Aware Resolution
	//
	// Strategy: Query halt(Entity, Reason, Tier) first (arity 3).
	// Rules loaded with tier annotations produce tier-attributed facts.
	// If no arity-3 matches, fall back to halt(Entity, Reason) (arity 2)
	// for backward compatibility — these get TierUnknown (lowest priority).
	//
	// This prevents a T3-authored halt from overriding a T0 halt
	// when both fire on the same envelope.

	var violationMsg, ruleID string

	type haltMatch struct {
		reason string
		tier   core.Tier
	}

	var matches []haltMatch

	if e.logger != nil {
		e.logger.Debug("Evaluating Gate Facts", "count", len(facts))
		for _, f := range facts {
			e.logger.Debug("Fact", "f", f.String())
		}
	}

	// Step 6a: Query arity-3 form: halt(Entity, Reason, Tier)
	// This is the tier-aware path — rules that emit halt/3 carry explicit tier.
	queryHalt3 := fmt.Sprintf("%s(\"%s\", Reason, Tier)", core.PredHalt, entityID)
	err = e.runtime.QueryWithSolutions(ctx, facts, queryHalt3, func(solution map[string]any) error {
		reason, rOk := solution["Reason"].(string)
		tierStr, tOk := solution["Tier"].(string)
		if !rOk {
			return nil
		}
		tier := core.Tier(tierStr)
		if !tOk || tier == "" {
			tier = core.TierUnknown
		}
		matches = append(matches, haltMatch{reason: reason, tier: tier})
		return nil
	})

	if err != nil {
		if e.logger != nil {
			e.logger.Debug("halt/3 query failed, falling back to halt/2", "error", err)
		}
	}

	// Step 6b: If no arity-3 matches, query arity-2 form for backward compat.
	// These matches get TierUnknown — they lose to any tier-attributed match.
	if len(matches) == 0 {
		queryHalt2 := fmt.Sprintf("%s(\"%s\", Reason)", core.PredHalt, entityID)
		err = e.runtime.QueryWithSolutions(ctx, facts, queryHalt2, func(solution map[string]any) error {
			if reason, ok := solution["Reason"].(string); ok {
				matches = append(matches, haltMatch{reason: reason, tier: core.TierUnknown})
			}
			return nil
		})

		if err != nil {
			if e.logger != nil {
				e.logger.Debug("failed to execute halt query", "error", err)
			}
			return nil, fmt.Errorf("halt query error: %w", err)
		}
	}

	// Resolve by tier: pick the match with the highest-priority tier
	if len(matches) > 0 {
		best := matches[0]
		for _, m := range matches[1:] {
			if tierPriority(m.tier) < tierPriority(best.tier) {
				best = m
			}
		}
		violationMsg = best.reason

		// Try to find rule ID if available
		qErr := e.runtime.QueryWithSolutions(ctx, facts, "violation_rule(ID)", func(solution map[string]any) error {
			if id, ok := solution["ID"].(string); ok {
				ruleID = id
				return ErrSolutionFound
			}
			return nil
		})
		if errors.Is(qErr, ErrSolutionFound) {
			qErr = nil
		}
		if qErr != nil && e.logger != nil {
			e.logger.Warn("failed to query violation rule ID", "error", qErr)
		}

		if e.logger != nil {
			e.logger.Debug("gate violation detected", "action", actionName, "msg", violationMsg, "rule_id", ruleID, "tier", best.tier, "matches", len(matches))
		}

		// Build audit trail from the collected matches.
		trail := core.NewAuditTrail("manglekit-engine", fmt.Sprintf("halt(\"%s\", ...)", entityID))
		for _, m := range matches {
			trail.AddRule("halt", fmt.Sprintf("halt(\"%s\", \"%s\", \"%s\")", entityID, m.reason, m.tier),
			 getSourceFileForPredicate("halt"), "halt", m.tier, nil)
		}
		trail.MatchedCount = len(matches)

		return trail, &core.AlignmentError{Message: violationMsg, RuleID: ruleID}
	}

	// Priority 2: deny(Entity) (Backward Compatibility)
	// Map legacy "deny" to core.PredHalt ("halt")
	queryDeny := fmt.Sprintf("%s(\"%s\")", core.PredHalt, entityID)
	denied, err := e.runtime.ExecuteQuery(ctx, facts, queryDeny)
	if err != nil {
		if e.logger != nil {
			e.logger.Debug("policy evaluation failed (deny check)", "error", err)
		}
		return nil, fmt.Errorf("policy evaluation error: %w", err)
	}

	if denied {
		// Query: violation_msg(Msg)
		qErr := e.runtime.QueryWithSolutions(ctx, facts, fmt.Sprintf("%s(Msg)", core.PredViolation), func(solution map[string]any) error {
			if msg, ok := solution["Msg"].(string); ok {
				violationMsg = msg
				return ErrSolutionFound
			}
			return nil
		})
		if errors.Is(qErr, ErrSolutionFound) {
			qErr = nil
		}
		if qErr != nil && e.logger != nil {
			e.logger.Warn("failed to query violation message", "error", qErr)
		}

		// Query: violation_rule(ID)
		qErr = e.runtime.QueryWithSolutions(ctx, facts, "violation_rule(ID)", func(solution map[string]any) error {
			if id, ok := solution["ID"].(string); ok {
				ruleID = id
				return ErrSolutionFound
			}
			return nil
		})
		if errors.Is(qErr, ErrSolutionFound) {
			qErr = nil
		}
		if qErr != nil && e.logger != nil {
			e.logger.Warn("failed to query violation rule ID", "error", qErr)
		}

		if e.logger != nil {
			e.logger.Debug("gate violation detected (deny)", "action", actionName, "msg", violationMsg, "rule_id", ruleID)
		}
		return nil, &core.AlignmentError{Message: violationMsg, RuleID: ruleID}
	}

	return nil, nil
}

// CheckRequirement queries: requires("req_id", "capability")
func (e *PolicyEngine) CheckRequirement(ctx context.Context, input core.Envelope, reqName string) (bool, error) {
	if e.runtime == nil {
		return false, nil
	}

	// 1. Convert Input to Facts using the PRIVATE helper
	// Signature: toMangleFacts(entityID, payload, contentType)
	facts, err := toMangleFacts(core.EntityInput, input.Payload, input.ContentType)
	if err != nil {
		return false, fmt.Errorf("fact conversion failed: %w", err)
	}

	// 2. Construct Query
	// Query format: requires("Req", "memory")
	query := fmt.Sprintf(`requires("%s", "%s")`, core.EntityInput, reqName)

	// 3. Execute Query
	// Returns (bool, error) directly as per current engine design
	return e.ExecuteQuery(ctx, facts, query)
}

// EvaluateSteering executes "Steering Policies" which determine what to do next.
// Unlike Assess/Reflect (which are binary Proceed/Infeasible), Steering returns decisions like "Retry" or "Route".
//
// Logic Priority:
//  1. Correction (Retry): If `retry(Hint)` is derived, we return `RETRY` with the hint.
//  2. Routing (Route): If `route(Target)` is derived, we return `ROUTE` with the target.
//  3. Default: `PROCEED` (Proceed as normal).
//
// Parameters:
//   - ctx: The execution context.
//   - input: The input envelope.
//
// Returns:
//   - decision: The decision string (e.g., "RETRY", "ROUTE", "PROCEED").
//   - metadata: A map containing steering details (e.g., {"feedback": "hint"}).
//   - error: An error if evaluation fails.
func (e *PolicyEngine) EvaluateSteering(ctx context.Context, input core.Envelope) (string, map[string]string, error) {
	decision := core.DecisionProceed
	metadata := make(map[string]string)

	if e.runtime == nil || e.runtime.programInfo == nil {
		return decision, metadata, nil
	}

	// Convert the input payload to Mangle facts
	// We use "Req" as the entity ID, consistent with Assess
	facts, err := toMangleFacts(core.EntityInput, input.Payload, input.ContentType)
	if err != nil {
		if e.logger != nil {
			e.logger.Debug("failed to convert input to facts for steering", "error", err)
		}
		return decision, metadata, fmt.Errorf("failed to convert input to facts: %w", err)
	}

	// [NEW] Inject Explicit Facts from Envelope
	for _, factStr := range input.Facts {
		atom, err := parse.Atom(factStr)
		if err != nil {
			if e.logger != nil {
				e.logger.Error("failed to parse envelop fact", "fact", factStr, "error", err)
			}
			return decision, metadata, fmt.Errorf("envelope fact parsing error: %w", err)
		}
		facts = append(facts, atom)
	}

	// Inject Metadata facts: meta(Key, Value) and attempt(N)
	for k, v := range input.Metadata {
		safeK := escapeString(k)
		vStr := fmt.Sprintf("%v", v)
		safeV := escapeString(vStr)
		// meta("key", "val")
		metaFact := fmt.Sprintf("meta(\"%s\", \"%s\")", safeK, safeV)
		if atom, err := parse.Atom(metaFact); err == nil {
			facts = append(facts, atom)
		}

		// attempt(N) from retry_count
		if k == "retry_count" {
			attemptFact := fmt.Sprintf("attempt(%s)", vStr)
			if atom, err := parse.Atom(attemptFact); err == nil {
				facts = append(facts, atom)
			}
		}
	}

	// 1. Check Correction (Retry) — Tier-Aware
	type retryMatch struct {
		hint string
		tier core.Tier
	}
	var retryMatches []retryMatch

	// Query arity-3 first: retry(Entity, Hint, Tier)
	retryQ3 := fmt.Sprintf("%s(\"%s\", Hint, Tier)", core.PredRetry, core.EntityInput)
	if err := e.runtime.QueryWithSolutions(ctx, facts, retryQ3, func(solution map[string]any) error {
		if hint, ok := solution["Hint"].(string); ok {
			tier := core.TierUnknown
			if t, ok := solution["Tier"].(string); ok && t != "" {
				tier = core.Tier(t)
			}
			retryMatches = append(retryMatches, retryMatch{hint: hint, tier: tier})
		}
		return nil
	}); err != nil {
		return decision, metadata, fmt.Errorf("retry query failed: %w", err)
	}

	// Fallback to arity-2: retry(Hint)
	if len(retryMatches) == 0 {
		if err := e.runtime.QueryWithSolutions(ctx, facts, fmt.Sprintf("%s(Hint)", core.PredRetry), func(solution map[string]any) error {
			if hint, ok := solution["Hint"].(string); ok {
				retryMatches = append(retryMatches, retryMatch{hint: hint, tier: core.TierUnknown})
			}
			return nil
		}); err != nil {
			return decision, metadata, fmt.Errorf("retry query failed: %w", err)
		}
	}

	// 2. Check Routing — Tier-Aware
	type routeMatch struct {
		target string
		tier   core.Tier
	}
	var routeMatches []routeMatch

	// Query arity-3 first: route(Entity, Target, Tier)
	routeQ3 := fmt.Sprintf("%s(\"%s\", Target, Tier)", core.PredRoute, core.EntityInput)
	if err := e.runtime.QueryWithSolutions(ctx, facts, routeQ3, func(solution map[string]any) error {
		if target, ok := solution["Target"].(string); ok {
			tier := core.TierUnknown
			if t, ok := solution["Tier"].(string); ok && t != "" {
				tier = core.Tier(t)
			}
			routeMatches = append(routeMatches, routeMatch{target: target, tier: tier})
		}
		return nil
	}); err != nil {
		return decision, metadata, fmt.Errorf("route query failed: %w", err)
	}

	// Fallback to arity-2: route(Target)
	if len(routeMatches) == 0 {
		if err := e.runtime.QueryWithSolutions(ctx, facts, fmt.Sprintf("%s(Target)", core.PredRoute), func(solution map[string]any) error {
			if target, ok := solution["Target"].(string); ok {
				routeMatches = append(routeMatches, routeMatch{target: target, tier: core.TierUnknown})
			}
			return nil
		}); err != nil {
			return decision, metadata, fmt.Errorf("route query failed: %w", err)
		}
	}

	// 3. Resolve by tier priority.
	// Retry always outranks route (correction before redirection).
	// Within each category, the highest-tier match wins.
	if len(retryMatches) > 0 {
		best := retryMatches[0]
		for _, m := range retryMatches[1:] {
			if tierPriority(m.tier) < tierPriority(best.tier) {
				best = m
			}
		}
		return core.DecisionRetry, map[string]string{core.KeyFeedback: best.hint}, nil
	}

	if len(routeMatches) > 0 {
		best := routeMatches[0]
		for _, m := range routeMatches[1:] {
			if tierPriority(m.tier) < tierPriority(best.tier) {
				best = m
			}
		}
		return core.DecisionRoute, map[string]string{core.KeyNextStep: best.target}, nil
	}

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
	if e.queryTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, e.queryTimeout)
		defer cancel()
	}

	if e.tracer == nil {
		return e.runtime.ExecuteQuery(ctx, facts, queryStr)
	}

	ctx, span := e.tracer.Start(ctx, "Datalog.ExecuteQuery")
	defer span.End()

	span.SetAttributes(map[string]any{"datalog.query": queryStr})

	res, err := e.runtime.ExecuteQuery(ctx, facts, queryStr)
	if err != nil {
		span.RecordError(err)
		return false, err
	}

	span.SetAttributes(map[string]any{"datalog.result": res})
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

	// Apply query timeout if configured
	queryCtx := ctx
	if e.queryTimeout > 0 {
		var cancel context.CancelFunc
		queryCtx, cancel = context.WithTimeout(ctx, e.queryTimeout)
		defer cancel()
	}

	if e.tracer != nil {
		var span core.Span
		queryCtx, span = e.tracer.Start(queryCtx, "Datalog.Query")
		defer span.End()
		span.SetAttributes(map[string]any{"datalog.query": queryStr})
	}

	err := e.runtime.QueryWithSolutions(queryCtx, atomFacts, queryStr, func(solution map[string]any) error {
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

// QueryWithAuditResult contains the query results along with the audit trail.
type QueryWithAuditResult struct {
	Results    []map[string]string
	AuditTrail *core.AuditTrail
}

// QueryWithAudit executes a Datalog query and returns results along with an audit trail.
// The audit trail explains which rules were matched and from which tier they originated.
func (e *PolicyEngine) QueryWithAudit(ctx context.Context, facts []string, queryStr string) (*QueryWithAuditResult, error) {
	startTime := time.Now()

	engineID := "manglekit-engine"
	auditTrail := core.NewAuditTrail(engineID, queryStr)

	var results []map[string]string

	// Parse temporary facts
	var atomFacts []ast.Atom
	for _, f := range facts {
		atom, err := parse.Atom(f)
		if err != nil {
			return nil, fmt.Errorf("failed to parse fact '%s': %w", f, err)
		}
		atomFacts = append(atomFacts, atom)
	}

	auditTrail.FactCount = len(atomFacts)

	// Apply query timeout if configured
	queryCtx := ctx
	if e.queryTimeout > 0 {
		var cancel context.CancelFunc
		queryCtx, cancel = context.WithTimeout(ctx, e.queryTimeout)
		defer cancel()
	}

	if e.tracer != nil {
		var span core.Span
		queryCtx, span = e.tracer.Start(queryCtx, "Datalog.QueryWithAudit")
		defer span.End()
		span.SetAttributes(map[string]any{"datalog.query": queryStr})
	}

	err := e.runtime.QueryWithSolutions(queryCtx, atomFacts, queryStr, func(solution map[string]any) error {
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

		// Extract rule information from the solution
		// This is a simplified version - in production, you'd inspect the proof
		extractRuleInference(auditTrail, solution, queryStr)

		return nil
	})

	if err != nil {
		return nil, err
	}

	auditTrail.MatchedCount = len(results)
	auditTrail.LatencyMs = time.Since(startTime).Milliseconds()

	return &QueryWithAuditResult{
		Results:    results,
		AuditTrail: auditTrail,
	}, nil
}

// extractRuleInference extracts rule inference information from the query solution.
// This is a simplified implementation that tries to identify which predicates matched.
func extractRuleInference(auditTrail *core.AuditTrail, solution map[string]any, queryStr string) {
	// Extract predicate names from the query
	// The query might look like: "can_execute(Agent, Action)" or "allow(Subject, Object)"
	// We need to identify which predicates in the knowledge base matched

	// For now, we'll create a basic inference entry
	// In a full implementation, you'd use mangle-go's proof/explanation API

	predicate := extractPredicateFromQuery(queryStr)
	if predicate == "" {
		return
	}

	// Try to determine the rule name and tier from the predicate
	ruleName := predicate
	tier := determineTierFromPredicate(predicate)
	sourceFile := getSourceFileForPredicate(predicate)

	// Convert solution values to string bindings
	bindings := make(map[string]string)
	for k, v := range solution {
		bindings[k] = fmt.Sprintf("%v", v)
	}

	definition := fmt.Sprintf("%s with bindings %v", predicate, bindings)

	auditTrail.AddRule(ruleName, definition, sourceFile, predicate, tier, bindings)
}

// extractPredicateFromQuery extracts the main predicate from a Datalog query string.
func extractPredicateFromQuery(queryStr string) string {
	// Simple extraction - take the first term before parenthesis
	queryStr = strings.TrimSpace(queryStr)

	// Handle queries like "can_execute(Agent, Action)" or "allow(X)"
	if idx := strings.Index(queryStr, "("); idx > 0 {
		return strings.TrimSpace(queryStr[:idx])
	}

	// Handle simple atom queries like "valid" or "allowed"
	return queryStr
}

// TierMapping maps predicate prefixes to governance tiers.
var TierMapping = map[string]core.Tier{
	"allow":           core.TierT1_Governance,
	"deny":            core.TierT1_Governance,
	"may_read":        core.TierT1_Governance,
	"may_write":       core.TierT1_Governance,
	"may_exec":        core.TierT1_Governance,
	"can_execute":     core.TierT2_Playbook,
	"can_access":      core.TierT2_Playbook,
	"requires":        core.TierT2_Playbook,
	"retry_for":       core.TierT2_Playbook,
	"validation_rule": core.TierT2_Playbook,
	"prompt_template": core.TierT2_Playbook,
	"workflow_node":   core.TierT2_Playbook,
	"workflow_edge":   core.TierT2_Playbook,
	"agent_role":      core.TierT2_Playbook,
	"role_capability": core.TierT2_Playbook,
	"task_requires":   core.TierT2_Playbook,
}

// TierSourceFiles maps tiers to their source files.
var TierSourceFiles = map[core.Tier]string{
	core.TierT0_Axiom:      "axioms.dl",
	core.TierT1_Governance: "governance.dl",
	core.TierT2_Playbook:   "registry.dl",
	core.TierT3_User:       "user-input.dl",
	core.TierUnknown:       "unknown",
}

// determineTierFromPredicate determines the governance tier from a predicate name.
func determineTierFromPredicate(predicate string) core.Tier {
	// Check exact match first
	if tier, ok := TierMapping[predicate]; ok {
		return tier
	}

	// Check prefix match
	for prefix, tier := range TierMapping {
		if strings.HasPrefix(predicate, prefix) {
			return tier
		}
	}

	return core.TierUnknown
}

// getSourceFileForPredicate returns the likely source file for a predicate.
func getSourceFileForPredicate(predicate string) string {
	tier := determineTierFromPredicate(predicate)
	return TierSourceFiles[tier]
}
