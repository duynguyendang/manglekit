package core

import "context"

// Evaluator: The Mangle Logic Engine.
// It defines the contract for policy execution, validation, and steering.
type Evaluator interface {
	// Assess evaluates the policy for a given input (General purpose).
	Assess(ctx context.Context, input Envelope) (Decision, error)

	// Authorize performs the Pre-Check phase (input validation).
	Authorize(ctx context.Context, actionMeta ActionMetadata, input Envelope) error

	// Validate performs the Post-Check phase (output validation).
	Validate(ctx context.Context, actionMeta ActionMetadata, output Envelope) (Envelope, error)

	// EvaluateSteering determines the next step (Retry/Route) based on the output.
	EvaluateSteering(ctx context.Context, input Envelope) (string, map[string]string, error)

	// GetActionConfig queries the engine for dynamic configuration parameters.
	GetActionConfig(ctx context.Context, input Envelope) (map[string]string, error)

	// LoadPolicy loads policy rules from a source string or file content.
	LoadPolicy(ctx context.Context, source string) error

	// LoadFacts injects dynamic facts into the engine.
	LoadFacts(facts []string) error

	// RegisterAction registers action metadata for discovery/planning.
	RegisterAction(meta ActionMetadata) error

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
