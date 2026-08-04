package core

import "context"

// Assessor evaluates policies against envelopes: the pre-check, post-check,
// steering, and configuration queries.
type Assessor interface {
	// AssessPlan evaluates the policy for a given input (General purpose).
	// It returns a Decision struct with the outcome.
	AssessPlan(ctx context.Context, input Envelope) (Decision, error)

	// Assess performs the Pre-Check phase (input validation).
	Assess(ctx context.Context, actionMeta ActionMetadata, input Envelope) error

	// Reflect evaluates the outcome of an action (Post-Check).
	Reflect(ctx context.Context, actionMeta ActionMetadata, output Envelope) (Envelope, error)

	// EvaluateSteering determines the next step (Retry/Route) based on the output.
	EvaluateSteering(ctx context.Context, input Envelope) (string, map[string]string, error)

	// GetActionConfig queries the engine for dynamic configuration parameters.
	GetActionConfig(ctx context.Context, input Envelope) (map[string]string, error)

	// CheckRequirement checks if a capability is needed. e.g., requires(Req, "memory").
	CheckRequirement(ctx context.Context, input Envelope, reqName string) (bool, error)
}

// ExternalPredicateRegistrar registers Go callbacks as external Datalog
// predicates. Implemented by the policy engine and the raw Mangle runtime.
type ExternalPredicateRegistrar interface {
	RegisterExternalPredicate(name string, fn func(ctx context.Context, inputs []any) ([][]any, error)) error
}

// PolicyLoader loads policies and facts into the engine.
type PolicyLoader interface {
	ExternalPredicateRegistrar

	// LoadPolicy loads policy rules incrementally from a source string.
	LoadPolicy(ctx context.Context, source string) error

	// LoadFromSource loads a Datalog program from a raw string, replacing
	// existing state and auto-emitting external predicate declarations.
	LoadFromSource(ctx context.Context, source string) error

	// LoadGherkinPolicy loads a Gherkin feature file and compiles it to Datalog.
	LoadGherkinPolicy(ctx context.Context, featureContent string) error

	// LoadFacts injects dynamic facts into the engine.
	LoadFacts(ctx context.Context, facts []string) error

	// RegisterAction registers action metadata for discovery/planning.
	RegisterAction(meta ActionMetadata) error
}

// Querier executes Datalog queries and returns all matching solutions.
// It is used by the planner to reason about goals and generate action sequences.
type Querier interface {
	Query(ctx context.Context, facts []string, queryStr string) ([]map[string]string, error)
}

// Evaluator: The Mangle Logic Engine.
// Full engine contract composed of Assessor, PolicyLoader, and Querier.
// Consumers that need only part of the surface should depend on the
// narrower interface instead.
type Evaluator interface {
	Assessor
	PolicyLoader
	Querier

	// Logger returns the engine's logger.
	Logger() Logger
}

// PreProcessor: Fast/Stateless checks (CEL/Expr).
type PreProcessor interface {
	Process(ctx context.Context, input Envelope) (map[string]any, error)
}

// RiskEngine: specialized interface for calculating risk.
type RiskEngine interface {
	CalculateRisk(ctx context.Context, input Envelope) (float64, error)
}

// ResourceMonitor: Cost & Rate Limiting.
type ResourceMonitor interface {
	CountTokens(ctx context.Context, text string) (int, error)
	CheckBudget(ctx context.Context, key string, cost int) (bool, error)
}
