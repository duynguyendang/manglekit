package east

import (
	"context"
	"fmt"
	"time"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/sdk/ooda"
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
// Includes entropy measurement, fast-path routing, Datalog validation, and
// Teacher-Student retry.
//
// Flow: Observe → Orient → Plan → [EAST routes] → Decide(opt) → Act →
// Validate + EAST post-act.
//
// EAST post-act behaviors:
//   - Validate: Datalog rule check (quality_gate, validation_rule, generic_pattern)
//   - Route: E decides HALT/RETRY/ACCEPT
//   - Retry: Teacher-Student loop with feedback injection (max 3)
func RunOODAEAST(ctx context.Context, frame *ooda.CognitiveFrame) (*ooda.CognitiveFrame, error) {
	ctx, cancel := context.WithTimeout(ctx, frame.Timeout)
	defer cancel()

	if frame.PhaseDurations == nil {
		frame.PhaseDurations = make(map[ooda.Phase]time.Duration)
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
	var path ooda.ExecutionPath
	if frame.ReasoningPort != nil {
		path = SteerKB(ctx, &frame.EAST, frame, frame.ReasoningPort)
	} else {
		path = Steer(&frame.EAST, frame)
	}

	// 5. Decide (always run — fast-path only skips EAST steering enrichment)
	if path == ooda.PathFast {
		core.LoggerFromContext(ctx).Debug("EAST fast-path: simplified Decide",
			"entropy", frame.EAST.Entropy, "trust", frame.EAST.TrustTier)
	}
	if err := decideWithEAST(ctx, frame); err != nil {
		return frame, fmt.Errorf("decide failed: %w", err)
	}

	// 6. Act (execute with authorization)
	if err := ooda.Act(ctx, frame); err != nil {
		return frame, fmt.Errorf("act failed: %w", err)
	}

	// 7. EAST post-act behaviors (validate + route + retry)
	return eastPostAct(ctx, frame)
}

// planWithEAST executes the planning phase with EAST steering.
func planWithEAST(ctx context.Context, frame *ooda.CognitiveFrame) error {
	start := time.Now()
	frame.Phase = ooda.PhasePlan

	if planner, ok := frame.Brain.(ooda.Planner); ok {
		if err := planner.Plan(ctx, frame); err != nil {
			return fmt.Errorf("planning failed: %w", err)
		}
	}

	frame.PhaseDurations[ooda.PhasePlan] = time.Since(start)
	return nil
}

// observeWithSaliency captures input and measures saliency.
func observeWithSaliency(ctx context.Context, frame *ooda.CognitiveFrame) error {
	start := time.Now()
	frame.Phase = ooda.PhaseObserve

	if frame.Input == "" {
		return fmt.Errorf("empty input")
	}

	frame.Context = append(frame.Context, ooda.Atom{
		Predicate: "raw_input",
		Subject:   frame.ID.String(),
		Object:    frame.Input,
		Weight:    0.3,
	})

	frame.EAST.Saliency = MeasureSaliency(frame.Input)

	frame.PhaseDurations[ooda.PhaseObserve] = time.Since(start)
	return nil
}

// orientWithEAST loads context and calculates the full EAST state.
func orientWithEAST(ctx context.Context, frame *ooda.CognitiveFrame) error {
	start := time.Now()
	frame.Phase = ooda.PhaseOrient

	if err := ooda.Orient(ctx, frame); err != nil {
		return err
	}

	// E: Entropy
	conflictCount := DetectConflicts(frame.Context, frame.AttentionSink)
	totalRules := len(frame.Context) + len(frame.AttentionSink)
	frame.EAST.Entropy = CalculateEntropy(conflictCount, totalRules)

	// A: Activity
	frame.EAST.Activity = CalculateActivity(frame.Context)

	// T: Trust
	if frame.Decision != nil && frame.AuditTrail != nil {
		frame.EAST.TrustTier = ooda.ClassifyViolations(frame.AuditTrail)
	} else {
		if len(frame.AttentionSink) > 0 {
			frame.EAST.TrustTier = ooda.Tier0Kernel
		} else {
			frame.EAST.TrustTier = ooda.Tier2AI
		}
	}

	CalculateMagnitude(&frame.EAST)

	if len(frame.EAST.Activity) > 0 {
		ooda.PruneColdAtoms(frame, frame.EAST.Activity)
	}

	execObj := ooda.NewExecutionObject(frame)
	if frame.RawContext == nil {
		frame.RawContext = make(map[string]any)
	}
	frame.RawContext["execution_object"] = execObj

	frame.PhaseDurations[ooda.PhaseOrient] = time.Since(start)
	return nil
}

// decideWithEAST evaluates policy with EAST steering applied.
func decideWithEAST(ctx context.Context, frame *ooda.CognitiveFrame) error {
	start := time.Now()
	frame.Phase = ooda.PhaseDecide

	if frame.Brain == nil {
		return nil
	}

	if ShouldInjectParadox(&frame.EAST) {
		if frame.RawContext == nil {
			frame.RawContext = make(map[string]any)
		}
		frame.RawContext["east_inject_paradox"] = true
		frame.RawContext["east_temperature"] = Temperature(&frame.EAST)
		frame.RawContext["east_steering_magnitude"] = frame.EAST.SteeringMagnitude
	}

	decision, err := frame.Brain.Evaluate(ctx, frame)
	if err != nil {
		frame.EAST.LogicSuccess = max(0, frame.EAST.LogicSuccess-0.1)
		return fmt.Errorf("brain evaluation failed: %w", err)
	}

	frame.EAST.LogicSuccess = min(1, frame.EAST.LogicSuccess+0.05)

	frame.Decision = decision
	if decision != nil {
		frame.AuditTrail = decision.AuditTrail
	}

	frame.PhaseDurations[ooda.PhaseDecide] = time.Since(start)
	return nil
}

// eastPostAct implements the EAST post-act behaviors: validate → route → retry.
func eastPostAct(ctx context.Context, frame *ooda.CognitiveFrame) (*ooda.CognitiveFrame, error) {
	start := time.Now()
	frame.Phase = ooda.PhasePostAct
	maxRetries := frame.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	for attempt := 0; attempt < maxRetries; attempt++ {
		frame.RetryCount = attempt + 1

		audit, err := ooda.ValidateAgainstRules(ctx, frame)
		if err != nil {
			return frame, fmt.Errorf("validation failed: %w", err)
		}

		frame.EAST.Entropy = UpdateEntropy(frame.EAST.Entropy, violationsCount(audit), totalRulesCount(frame))

		reason := ooda.ShouldTerminate(audit, frame.RetryCount, maxRetries, frame.EAST.Entropy)
		switch reason {
		case "t0_violation":
			frame.Status = ooda.VerifyStatusFailed
			frame.PhaseDurations[ooda.PhasePostAct] = time.Since(start)
			return frame, ErrT0Violation
		case "max_iterations":
			frame.Status = ooda.VerifyStatusFailed
			frame.PhaseDurations[ooda.PhasePostAct] = time.Since(start)
			return frame, ErrMaxRefinementIterations
		case "chaos_threshold":
			frame.Status = ooda.VerifyStatusFailed
			frame.PhaseDurations[ooda.PhasePostAct] = time.Since(start)
			return frame, ErrChaosThreshold
		}

		if audit.Pass {
			frame.Status = ooda.VerifyStatusPassed
			if frame.Memory != nil {
				frame.Memory.Commit(ctx, frame)
			}
			frame.PhaseDurations[ooda.PhasePostAct] = time.Since(start)
			return frame, nil
		}

		if frame.EAST.Entropy < 0.7 && audit.ViolationTier == ooda.Tier3User {
			frame.Status = ooda.VerifyStatusWarning
			if frame.Memory != nil {
				frame.Memory.Commit(ctx, frame)
			}
			frame.PhaseDurations[ooda.PhasePostAct] = time.Since(start)
			return frame, nil
		}

		feedback := ooda.NewRefinementContext(audit, frame.Draft, attempt+1)
		if frame.RawContext == nil {
			frame.RawContext = make(map[string]any)
		}
		frame.RawContext["refinement_feedback"] = feedback

		core.LoggerFromContext(ctx).Warn("EAST retry",
			"attempt", attempt+1, "max", maxRetries, "entropy", frame.EAST.Entropy, "reason", audit.ViolationTier)

		if err := decideWithEAST(ctx, frame); err != nil {
			return frame, fmt.Errorf("re-decide failed: %w", err)
		}
		if err := ooda.Act(ctx, frame); err != nil {
			return frame, fmt.Errorf("re-act failed: %w", err)
		}
	}

	frame.Status = ooda.VerifyStatusFailed
	frame.PhaseDurations[ooda.PhasePostAct] = time.Since(start)
	return frame, ErrMaxRefinementIterations
}

// violationsCount returns the number of violations from an AuditResult.
func violationsCount(audit *ooda.AuditResult) int {
	if audit == nil || audit.Pass {
		return 0
	}
	return 1
}

// totalRulesCount returns the total number of rules in the frame.
func totalRulesCount(frame *ooda.CognitiveFrame) int {
	return len(frame.Context) + len(frame.AttentionSink)
}
