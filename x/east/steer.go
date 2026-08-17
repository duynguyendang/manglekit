// Steering functions for the EAST (OODA v4) extension. These operate on the
// public ooda.EASTState / ooda.CognitiveFrame types so the core OODA frame
// model stays dependency-free of x/east.

package east

import (
	"context"
	"fmt"
	"math"

	"github.com/duynguyendang/manglekit/sdk/ooda"
	"github.com/duynguyendang/manglekit/sdk/ports"
)

// SteeringPolicy defines template keys used for prompt assembly when the
// agent reaches a specific cognitive pressure threshold.
type SteeringPolicy struct {
	MinMagnitude float64
	HeaderKey    string
	BodyKey      string
}

// SteeringRegistry holds the tiered policies for guiding LLM behavior under
// stress. Sorted by MinMagnitude descending; first match where magnitude >
// MinMagnitude wins.
var SteeringRegistry = []SteeringPolicy{
	{MinMagnitude: 0.8, HeaderKey: "prompt_header_paradox", BodyKey: "prompt_body_conservative"},
	{MinMagnitude: 0.5, HeaderKey: "prompt_header_strict", BodyKey: "prompt_body_structured"},
	{MinMagnitude: 0.0, HeaderKey: "prompt_header_default", BodyKey: "prompt_body_creative"},
}

// CalculateEAST computes the Entropic Activation Steering magnitude (P):
// P = exp(1 - LogicSuccess) / EntropyCoefficient.
func CalculateEAST(logicSuccess, entropyCoefficient float64) float64 {
	if entropyCoefficient <= 0.01 {
		entropyCoefficient = 0.01
	}
	if logicSuccess < 0.0 {
		logicSuccess = 0.0
	} else if logicSuccess > 1.0 {
		logicSuccess = 1.0
	}
	return math.Exp(1.0-logicSuccess) / entropyCoefficient
}

// GetSteeringPrompts selects the appropriate prompt templates based on the
// current EAST magnitude.
func GetSteeringPrompts(magnitude float64) (headerKey, bodyKey string) {
	for _, policy := range SteeringRegistry {
		if magnitude >= policy.MinMagnitude {
			return policy.HeaderKey, policy.BodyKey
		}
	}
	fallback := SteeringRegistry[len(SteeringRegistry)-1]
	return fallback.HeaderKey, fallback.BodyKey
}

// Steer determines the execution path based on EAST metrics.
func Steer(state *ooda.EASTState, frame *ooda.CognitiveFrame) ooda.ExecutionPath {
	if state.Entropy < 0.2 && state.TrustTier == ooda.Tier0Kernel && isKnownPattern(frame) {
		return ooda.PathFast
	}
	if state.Entropy > 0.7 || state.TrustTier == ooda.Tier3User || state.TrustTier == ooda.Tier2AI {
		return ooda.PathSlow
	}
	return ooda.PathStandard
}

// SteerKB determines the execution path by querying Datalog rules directly
// (14-east.dl). Falls back to in-memory Steer if ReasoningPort is nil.
func SteerKB(ctx context.Context, state *ooda.EASTState, frame *ooda.CognitiveFrame, reasoner ports.ReasoningPort) ooda.ExecutionPath {
	if reasoner == nil {
		return Steer(state, frame)
	}
	if results, err := reasoner.VerifyWithDatalog(ctx, fmt.Sprintf("fast_path(%s)", frame.ID)); err == nil && len(results) > 0 {
		return ooda.PathFast
	}
	if results, err := reasoner.VerifyWithDatalog(ctx, fmt.Sprintf("slow_path(%s)", frame.ID)); err == nil && len(results) > 0 {
		return ooda.PathSlow
	}
	return ooda.PathStandard
}

// CalculateMagnitude computes the steering formula: P = exp(1-L) / N.
func CalculateMagnitude(state *ooda.EASTState) float64 {
	if state.EntropyCoefficient == 0 {
		return 0
	}
	state.SteeringMagnitude = math.Exp(1.0-state.LogicSuccess) / state.EntropyCoefficient
	return state.SteeringMagnitude
}

// ShouldInjectParadox returns true if steering magnitude exceeds the
// configured paradox threshold. Falls back to the default (0.8) when unset.
// Callers should compute magnitude first (see CalculateMagnitude) — order in
// RunOODAEAST guarantees Magnitude is set before Decide.
func ShouldInjectParadox(state *ooda.EASTState) bool {
	threshold := state.ParadoxThreshold
	if threshold <= 0 {
		threshold = ooda.DefaultParadoxThreshold
	}
	return state.SteeringMagnitude > threshold
}

// Temperature returns the recommended LLM temperature based on steering
// magnitude.
func Temperature(state *ooda.EASTState) float64 {
	switch m := state.SteeringMagnitude; {
	case m < 0.3:
		return 0.9
	case m < 0.6:
		return 0.7
	case m < 0.8:
		return 0.4
	default:
		return 0.1
	}
}

// isKnownPattern checks if the frame's input matches a known project pattern.
func isKnownPattern(frame *ooda.CognitiveFrame) bool {
	if frame == nil {
		return false
	}
	for _, s := range frame.EAST.Saliency {
		if s == "known_pattern" {
			return true
		}
	}
	if v, ok := frame.RawContext["project_type"].(string); ok && v != "" {
		return true
	}
	return false
}
