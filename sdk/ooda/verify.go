package ooda

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/duynguyendang/manglekit/core"
	genkcore "github.com/firebase/genkit/go/core"
)

// ValidationError represents a validation failure with tier information.
type ValidationError struct {
	RuleName     string
	Tier         TrustTier
	Message      string
	ConflictPath string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("[%s] %s: %s", e.Tier, e.RuleName, e.Message)
}

// VerifySchema checks if the action result matches the expected output_schema.
// This is the OODA mode (tools) post-Act validation.
func VerifySchema(ctx context.Context, frame *CognitiveFrame) error {
	if frame.Decision == nil || frame.Decision.Action == nil {
		return nil // No action, nothing to verify
	}

	actionName := frame.Decision.Action.Name
	result := frame.ActionResult
	if result == nil {
		return nil
	}

	// Query Brain for output_schema if available
	if frame.Brain != nil {
		auditTrail, err := frame.Brain.Verify(ctx, frame)
		if err != nil {
			var schemaErr *genkcore.SchemaValidationError
			if errors.As(err, &schemaErr) {
				return &ValidationError{
					RuleName: "SCHEMA_VALIDATION",
					Tier:     Tier1Admin,
					Message:  fmt.Sprintf("action '%s' output failed schema validation: %v", actionName, schemaErr),
				}
			}
			return fmt.Errorf("schema verification failed: %w", err)
		}
		frame.AuditTrail = auditTrail
	}

	// Basic heuristic: check if result contains "error" or "failed"
	resultStr := fmt.Sprintf("%v", result)
	if strings.Contains(strings.ToLower(resultStr), "error") ||
		strings.Contains(strings.ToLower(resultStr), "failed") {
		return fmt.Errorf("action '%s' returned error-like result: %s", actionName, resultStr)
	}

	return nil
}

// ValidateAgainstRules performs Datalog-backed validation of the action result.
// This is the OODA+EAST mode (generation) post-Act validation.
// Returns the AuditResult with violation tier and matched rules.
func ValidateAgainstRules(ctx context.Context, frame *CognitiveFrame) (*AuditResult, error) {
	result := &AuditResult{
		Pass: true,
	}

	// 1. Query Brain.Verify for policy-based validation
	if frame.Brain != nil {
		auditTrail, err := frame.Brain.Verify(ctx, frame)
		if err != nil {
			return nil, fmt.Errorf("datalog validation failed: %w", err)
		}
		frame.AuditTrail = auditTrail

		// 2. Classify violations by tier
		worstTier := classifyViolations(auditTrail)
		result.ViolationTier = worstTier

		if len(auditTrail.MatchedRules) > 0 {
			// Check for violations (rules with severity)
			for _, rule := range auditTrail.MatchedRules {
				if isViolation(rule) {
					result.Pass = false
					result.ConflictPath = rule.SourceFile
					break
				}
			}
		}

		// 3. Set status based on worst tier
		switch worstTier {
		case Tier0Kernel:
			result.Pass = false
			result.EntropyDelta = 1.0 // Maximum entropy increase
		case Tier1Admin:
			result.Pass = false
			result.EntropyDelta = 0.5
		case Tier2AI:
			result.EntropyDelta = 0.2
		case Tier3User:
			result.EntropyDelta = 0.1
		}
	}

	return result, nil
}

// TieredVerify evaluates rules by trust tier (T0-T3).
// T0 violations → HALT, T1 → RETRY, T2 → WARNING, T3 → IGNORE.
func TieredVerify(ctx context.Context, frame *CognitiveFrame) (*AuditResult, error) {
	result := &AuditResult{Pass: true}

	if frame.Brain == nil {
		return result, nil
	}

	auditTrail, err := frame.Brain.Verify(ctx, frame)
	if err != nil {
		return nil, err
	}

	worstTier := classifyViolations(auditTrail)
	result.ViolationTier = worstTier

	// Find violations and classify by tier
	for _, rule := range auditTrail.MatchedRules {
		if isViolation(rule) {
			switch rule.Tier {
			case core.TierT0_Axiom:
				result.Pass = false
				result.EntropyDelta = 1.0
				result.ConflictPath = rule.SourceFile
				return result, nil // Stop at T0 — immediate HALT
			case core.TierT1_Governance:
				result.Pass = false
				result.EntropyDelta = 0.5
				result.ConflictPath = rule.SourceFile
				// Continue checking — T1 is retry, not halt
			case core.TierT2_Playbook:
				result.EntropyDelta = 0.2
				// WARNING — continue
			case core.TierT3_User:
				result.EntropyDelta = 0.1
				// IGNORE — continue
			}
		}
	}

	return result, nil
}

// ShouldTerminate checks if the refinement loop should terminate.
// Returns the termination reason if any, or empty string to continue.
func ShouldTerminate(audit *AuditResult, retryCount, maxRetries int, entropy float64) string {
	// T0 violation → never retry
	if audit.ViolationTier == Tier0Kernel {
		return "t0_violation"
	}

	// Max iterations
	if retryCount >= maxRetries {
		return "max_iterations"
	}

	// Chaos threshold (entropy too high)
	if entropy > 0.9 {
		return "chaos_threshold"
	}

	return "" // Continue
}

// BuildFeedback creates a RefinementContext from audit failures.
type RefinementContext struct {
	FailedRules   []string
	ConflictPath  string
	PreviousDraft any
	AttemptNumber int
}

// NewRefinementContext builds feedback for the Teacher-Student loop.
func NewRefinementContext(audit *AuditResult, draft any, attempt int) *RefinementContext {
	var rules []string
	if audit != nil {
		for _, r := range getMatchedRules(audit) {
			rules = append(rules, r)
		}
	}
	return &RefinementContext{
		FailedRules:   rules,
		ConflictPath:  audit.ConflictPath,
		PreviousDraft: draft,
		AttemptNumber: attempt,
	}
}

// classifyViolations returns the worst tier found in the audit trail.
func classifyViolations(trail *core.AuditTrail) TrustTier {
	if trail == nil || len(trail.MatchedRules) == 0 {
		return Tier3User
	}

	worst := Tier3User
	for _, rule := range trail.MatchedRules {
		if isViolation(rule) {
			tier := mapCoreTier(rule.Tier)
			if tierOrder(tier) < tierOrder(worst) {
				worst = tier
			}
		}
	}
	return worst
}

// isViolation checks if a rule represents a violation (not just a match).
func isViolation(rule core.RuleInference) bool {
	lower := strings.ToLower(rule.RuleName)
	return strings.Contains(lower, "halt") ||
		strings.Contains(lower, "deny") ||
		strings.Contains(lower, "violation") ||
		strings.Contains(lower, "prohibited")
}

// mapCoreTier converts core.Tier to ooda.TrustTier.
func mapCoreTier(t core.Tier) TrustTier {
	switch t {
	case core.TierT0_Axiom:
		return Tier0Kernel
	case core.TierT1_Governance:
		return Tier1Admin
	case core.TierT2_Playbook:
		return Tier2AI
	default:
		return Tier3User
	}
}

// tierOrder returns numeric ordering for tier comparison (lower = more severe).
func tierOrder(t TrustTier) int {
	switch t {
	case Tier0Kernel:
		return 0
	case Tier1Admin:
		return 1
	case Tier2AI:
		return 2
	case Tier3User:
		return 3
	default:
		return 4
	}
}

// getMatchedRules returns rule names from an AuditResult.
func getMatchedRules(audit *AuditResult) []string {
	// AuditResult stores violations but not full matched rules
	// For now, return the conflict path
	if audit.ConflictPath != "" {
		return []string{audit.ConflictPath}
	}
	return nil
}
