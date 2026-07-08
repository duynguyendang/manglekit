package logic

import (
	"math"
)

// SteeringPolicy defines template keys used for prompt assembly when the agent reaches
// a specific cognitive pressure threshold.
type SteeringPolicy struct {
	MinMagnitude float64
	HeaderKey    string // Prompt template key for the system header
	BodyKey      string // Prompt template key for the body instructions
}

// SteeringRegistry holds the tiered policies for guiding LLM behavior under stress.
// LLD 5.4: Sorted by MinMagnitude descending. First match where magnitude > MinMagnitude wins.
var SteeringRegistry = []SteeringPolicy{
	{
		MinMagnitude: 0.8,
		HeaderKey:    "prompt_header_paradox",
		BodyKey:      "prompt_body_conservative",
	},
	{
		MinMagnitude: 0.5,
		HeaderKey:    "prompt_header_strict",
		BodyKey:      "prompt_body_structured",
	},
	{
		MinMagnitude: 0.0,
		HeaderKey:    "prompt_header_default",
		BodyKey:      "prompt_body_creative",
	},
}

// CalculateEAST computes the Entropic Activation Steering magnitude (P).
// LLD 5.4 Formula: P = exp(1 - LogicSuccess) / EntropyCoefficient
func CalculateEAST(logicSuccess, entropyCoefficient float64) float64 {
	// Prevent division by zero
	if entropyCoefficient <= 0.01 {
		entropyCoefficient = 0.01
	}

	// LogicSuccess is bounded between 0.0 (total failure) and 1.0 (perfect compliance).
	if logicSuccess < 0.0 {
		logicSuccess = 0.0
	} else if logicSuccess > 1.0 {
		logicSuccess = 1.0
	}

	// e^(1 - L)
	activation := math.Exp(1.0 - logicSuccess)

	return activation / entropyCoefficient
}

// GetSteeringPrompts selects the appropriate prompt templates based on the current EAST magnitude.
func GetSteeringPrompts(magnitude float64) (headerKey, bodyKey string) {
	for _, policy := range SteeringRegistry {
		if magnitude >= policy.MinMagnitude {
			return policy.HeaderKey, policy.BodyKey
		}
	}
	// Fallback to lowest tier
	fallback := SteeringRegistry[len(SteeringRegistry)-1]
	return fallback.HeaderKey, fallback.BodyKey
}

// DefaultParadoxThreshold is the EAST magnitude above which cognitive paradox
// injection is triggered when no per-state threshold is configured.
const DefaultParadoxThreshold = 0.8
