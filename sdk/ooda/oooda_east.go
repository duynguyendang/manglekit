package ooda

import (
	"context"
	"fmt"
	"time"

	"github.com/duynguyendang/manglekit/core"
)

var (
	// ErrT0Violation is returned when a Tier 0 (kernel axiom) violation is detected.
	ErrT0Violation = fmt.Errorf("T0 violation: immediate halt")
	// ErrMaxRefinementIterations is returned when refinement exceeds max iterations.
	ErrMaxRefinementIterations = fmt.Errorf("max refinement iterations exceeded")
	// ErrChaosThreshold is returned when entropy exceeds chaos threshold (0.9).
	ErrChaosThreshold = fmt.Errorf("chaos threshold exceeded: entropy too high")
)

// RunOODAEAST executes the OODA loop with EAST steering for generation tasks.
// Includes entropy measurement, fast-path routing, Datalog validation, and Teacher-Student retry.
//
// Flow: Observe → Orient → Plan → [EAST routes] → Decide(opt) → Act → Validate + EAST post-act
//
// EAST post-act behaviors:
//   - Validate: Datalog rule check (quality_gate, validation_rule, generic_pattern)
//   - Route: E decides HALT/RETRY/ACCEPT
//   - Retry: Teacher-Student loop with feedback injection (max 3)
func RunOODAEAST(ctx context.Context, frame *CognitiveFrame) (*CognitiveFrame, error) {
	ctx, cancel := context.WithTimeout(ctx, frame.Timeout)
	defer cancel()

	if frame.PhaseDurations == nil {
		frame.PhaseDurations = make(map[Phase]time.Duration)
	}

	// 1. Observe (input validation + saliency measurement)
	if err := observeWithSaliency(ctx, frame); err != nil {
		return frame, fmt.Errorf("observe failed: %w", err)
	}

	// 2. Orient (synthesize context + calculate EAST state)
	if err := orientWithEAST(ctx, frame); err != nil {
		return frame, fmt.Errorf("orient failed: %w", err)
	}

	// 3. Plan (create explicit plan from planning phase)
	if err := planWithEAST(ctx, frame); err != nil {
		return frame, fmt.Errorf("plan failed: %w", err)
	}

	// 4. EAST routing decision
	// Use SteerKB() if ReasoningPort is available for Datalog-backed routing (14-east.dl)
	// Otherwise fall back to in-memory Steer()
	var path ExecutionPath
	if frame.ReasoningPort != nil {
		path = frame.EAST.SteerKB(ctx, frame, frame.ReasoningPort)
	} else {
		path = frame.EAST.Steer(frame)
	}

	// 5. Decide (always run — fast-path only skips EAST steering enrichment)
	if path == PathFast {
		core.LoggerFromContext(ctx).Debug("EAST fast-path: simplified Decide",
			"entropy", frame.EAST.Entropy, "trust", frame.EAST.TrustTier)
	}
	if err := decideWithEAST(ctx, frame); err != nil {
		return frame, fmt.Errorf("decide failed: %w", err)
	}

	// 6. Act (execute with authorization)
	if err := act(ctx, frame); err != nil {
		return frame, fmt.Errorf("act failed: %w", err)
	}

	// 7. EAST post-act behaviors (validate + route + retry)
	return eastPostAct(ctx, frame)
}

// planWithEAST executes the planning phase with EAST steering.
func planWithEAST(ctx context.Context, frame *CognitiveFrame) error {
	start := time.Now()
	frame.Phase = PhasePlan

	// Use Planner interface if Brain implements it
	if planner, ok := frame.Brain.(Planner); ok {
		if err := planner.Plan(ctx, frame); err != nil {
			return fmt.Errorf("planning failed: %w", err)
		}
	}

	frame.PhaseDurations[PhasePlan] = time.Since(start)
	return nil
}

// observeWithSaliency captures input and measures saliency.
func observeWithSaliency(ctx context.Context, frame *CognitiveFrame) error {
	start := time.Now()
	frame.Phase = PhaseObserve

	if frame.Input == "" {
		return fmt.Errorf("empty input")
	}

	// Capture input as atom
	frame.Context = append(frame.Context, Atom{
		Predicate: "raw_input",
		Subject:   frame.ID.String(),
		Object:    frame.Input,
		Weight:    0.3,
	})

	// Measure Saliency (S)
	frame.EAST.Saliency = MeasureSaliency(frame.Input)

	frame.PhaseDurations[PhaseObserve] = time.Since(start)
	return nil
}

// orientWithEAST loads context and calculates the full EAST state.
// Produces an ExecutionObject as the unified output of ORIENT.
func orientWithEAST(ctx context.Context, frame *CognitiveFrame) error {
	start := time.Now()
	frame.Phase = PhaseOrient

	// Load context from dual memory (same as existing orient)
	if err := orient(ctx, frame); err != nil {
		return err
	}

	// === Calculate EAST State ===

	// E: Entropy — detect conflicts between context and axioms
	conflictCount := DetectConflicts(frame.Context, frame.AttentionSink)
	totalRules := len(frame.Context) + len(frame.AttentionSink)
	frame.EAST.Entropy = CalculateEntropy(conflictCount, totalRules)

	// A: Activity — track which atoms are frequently accessed
	frame.EAST.Activity = CalculateActivity(frame.Context)

	// T: Trust — determine from decision source
	if frame.Decision != nil && frame.AuditTrail != nil {
		// Use the worst tier from matched rules
		worstTier := classifyViolations(frame.AuditTrail)
		frame.EAST.TrustTier = worstTier
	} else {
		// Default: if we have AttentionSink axioms, trust is high
		if len(frame.AttentionSink) > 0 {
			frame.EAST.TrustTier = Tier0Kernel
		} else {
			frame.EAST.TrustTier = Tier2AI
		}
	}

	// Calculate steering magnitude
	frame.EAST.CalculateMagnitude()

	// Prune cold atoms if activity tracking is available
	if len(frame.EAST.Activity) > 0 {
		PruneColdAtoms(frame, frame.EAST.Activity)
	}

	// Build ExecutionObject — the unified output of ORIENT
	execObj := NewExecutionObject(frame)
	if frame.RawContext == nil {
		frame.RawContext = make(map[string]any)
	}
	frame.RawContext["execution_object"] = execObj

	frame.PhaseDurations[PhaseOrient] = time.Since(start)
	return nil
}

// decideWithEAST evaluates policy with EAST steering applied.
func decideWithEAST(ctx context.Context, frame *CognitiveFrame) error {
	start := time.Now()
	frame.Phase = PhaseDecide

	if frame.Brain == nil {
		return nil
	}

	// Apply EAST steering to the decision
	// If paradox injection is active, the Brain should be informed
	if frame.EAST.ShouldInjectParadox() {
		if frame.RawContext == nil {
			frame.RawContext = make(map[string]any)
		}
		frame.RawContext["east_inject_paradox"] = true
		frame.RawContext["east_temperature"] = frame.EAST.Temperature()
		frame.RawContext["east_steering_magnitude"] = frame.EAST.SteeringMagnitude
	}

	// Evaluate
	decision, err := frame.Brain.Evaluate(ctx, frame)
	if err != nil {
		// Update EAST: failure decreases LogicSuccess
		frame.EAST.LogicSuccess = max(0, frame.EAST.LogicSuccess-0.1)
		return fmt.Errorf("brain evaluation failed: %w", err)
	}

	// Update EAST: success increases LogicSuccess
	frame.EAST.LogicSuccess = min(1, frame.EAST.LogicSuccess+0.05)

	frame.Decision = decision
	if decision != nil {
		frame.AuditTrail = decision.AuditTrail
	}

	frame.PhaseDurations[PhaseDecide] = time.Since(start)
	return nil
}

// eastPostAct implements the EAST post-act behaviors: validate → route → retry.
func eastPostAct(ctx context.Context, frame *CognitiveFrame) (*CognitiveFrame, error) {
	start := time.Now()
	frame.Phase = PhasePostAct
	maxRetries := frame.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	for attempt := 0; attempt < maxRetries; attempt++ {
		frame.RetryCount = attempt + 1

		// 1. Validate: check Datalog rules
		audit, err := ValidateAgainstRules(ctx, frame)
		if err != nil {
			return frame, fmt.Errorf("validation failed: %w", err)
		}

		// 2. Update EAST entropy
		frame.EAST.Entropy = UpdateEntropy(frame.EAST.Entropy, violationsCount(audit), totalRulesCount(frame))

		// 3. Check termination
		reason := ShouldTerminate(audit, frame.RetryCount, maxRetries, frame.EAST.Entropy)
		switch reason {
		case "t0_violation":
			frame.Status = VerifyStatusFailed
			frame.PhaseDurations[PhasePostAct] = time.Since(start)
			return frame, ErrT0Violation
		case "max_iterations":
			frame.Status = VerifyStatusFailed
			frame.PhaseDurations[PhasePostAct] = time.Since(start)
			return frame, ErrMaxRefinementIterations
		case "chaos_threshold":
			frame.Status = VerifyStatusFailed
			frame.PhaseDurations[PhasePostAct] = time.Since(start)
			return frame, ErrChaosThreshold
		}

		// 4. Accept if passed
		if audit.Pass {
			frame.Status = VerifyStatusPassed
			if frame.Memory != nil {
				frame.Memory.Commit(ctx, frame)
			}
			frame.PhaseDurations[PhasePostAct] = time.Since(start)
			return frame, nil
		}

		// 5. Accept with warning if entropy is low
		if frame.EAST.Entropy < 0.7 && audit.ViolationTier == Tier3User {
			frame.Status = VerifyStatusWarning
			if frame.Memory != nil {
				frame.Memory.Commit(ctx, frame)
			}
			frame.PhaseDurations[PhasePostAct] = time.Since(start)
			return frame, nil
		}

		// 6. RETRY: inject feedback, re-Decide, re-Act
		feedback := NewRefinementContext(audit, frame.Draft, attempt+1)
		if frame.RawContext == nil {
			frame.RawContext = make(map[string]any)
		}
		frame.RawContext["refinement_feedback"] = feedback

		core.LoggerFromContext(ctx).Warn("EAST retry",
			"attempt", attempt+1, "max", maxRetries, "entropy", frame.EAST.Entropy, "reason", audit.ViolationTier)

		// Re-Decide with feedback
		if err := decideWithEAST(ctx, frame); err != nil {
			return frame, fmt.Errorf("re-decide failed: %w", err)
		}

		// Re-Act
		if err := act(ctx, frame); err != nil {
			return frame, fmt.Errorf("re-act failed: %w", err)
		}
	}

	frame.Status = VerifyStatusFailed
	frame.PhaseDurations[PhasePostAct] = time.Since(start)
	return frame, ErrMaxRefinementIterations
}

// violationsCount returns the number of violations from an AuditResult.
func violationsCount(audit *AuditResult) int {
	if audit == nil || audit.Pass {
		return 0
	}
	return 1 // Simplified: one violation per audit
}

// totalRulesCount returns the total number of rules in the frame.
func totalRulesCount(frame *CognitiveFrame) int {
	return len(frame.Context) + len(frame.AttentionSink)
}


