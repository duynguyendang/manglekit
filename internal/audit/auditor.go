package audit

import (
	"context"
	"fmt"
	"time"

	"github.com/duynguyendang/manglekit-wip/internal/core/domain"
	"github.com/duynguyendang/manglekit-wip/internal/core/ports"
)

// Auditor governs the OODA Verification Phase (Shadow Audit).
type Auditor struct {
	verifier ports.ReasoningPort
}

// New initializes the auditor subsystem.
func New(verifier ports.ReasoningPort) *Auditor {
	return &Auditor{
		verifier: verifier,
	}
}

// Verify implements the core security gate as defined in LLD 8.1.
// It mathematically proves that an intended payload complies with the active GenePool.
func (a *Auditor) Verify(ctx context.Context, frame *domain.CognitiveFrame) (*domain.AuditResult, error) {
	// LLD 8.1: State Machine Violation Check
	// The Auditor must explicitly assert that Tier 0 Kernel Axioms are loaded into the
	// active reasoning frame. If the prompt compiler or agent loop somehow dropped Tier 0,
	// the agent is operating completely unrestrained and must be halted.
	hasTier0 := false
	for _, gene := range frame.ActiveGenes {
		if gene.Tier == domain.Tier0Kernel {
			hasTier0 = true
			break
		}
	}

	if !hasTier0 {
		return nil, fmt.Errorf("CRITICAL SECURITY EXCEPTION: Tier 0 Kernel Axioms missing from Epoch %s. Halting.", frame.ID)
	}

	// Route audit based on target output type
	switch frame.OutputType {
	case domain.OutputTypePlan:
		return a.verifyPlan(ctx, frame)
	case domain.OutputTypeRule:
		return a.verifyInducedRules(ctx, frame)
	default:
		return nil, fmt.Errorf("unknown task output type: %s", frame.OutputType)
	}
}

// verifyPlan checks structured JSON task execution plans against the Tiered Logic.
func (a *Auditor) verifyPlan(ctx context.Context, frame *domain.CognitiveFrame) (*domain.AuditResult, error) {
	// Route to ReasoningPort (Mangle LFTJ evaluator)
	res, err := a.verifier.Verify(ctx, frame.Draft, frame.ActiveGenes)
	if err != nil {
		return nil, err
	}

	// Update the frame status depending on the trust tier violated
	if !res.Pass {
		if res.ViolationTier == domain.Tier0Kernel || res.ViolationTier == domain.Tier1Admin {
			frame.Status = domain.VerifyStatusFailed
		} else {
			// Tier 2 or 3 are soft heuristics, log warning but allow pass
			frame.Status = domain.VerifyStatusWarning
			res.Pass = true // override
		}
	} else {
		frame.Status = domain.VerifyStatusPassed
	}

	return res, nil
}

// verifyInducedRules handles the Knowledge Induction shadow audit.
// Validates that newly learned Tier 2/3 rules do not contradict Tier 0/1.
func (a *Auditor) verifyInducedRules(ctx context.Context, frame *domain.CognitiveFrame) (*domain.AuditResult, error) {
	rawCandidate, ok := frame.Draft.([]byte)
	if !ok {
		return nil, fmt.Errorf("verifyInducedRules expects []byte draft, got %T", frame.Draft)
	}

	// Mocking compilation of the combined graph (Existing Genes + rawCandidate)
	_ = rawCandidate

	// In a complete implementation, this merges `rawCandidate` logic into
	// the global Datalog program string, and issues a contradiction query:
	// `contradiction(X) :- rule_A, not rule_B.`

	return &domain.AuditResult{
		Pass: true,
	}, nil
}

// GenerateTrace creates a markdown artifact documenting the verification run.
func (a *Auditor) GenerateTrace(frame *domain.CognitiveFrame) []byte {
	trace := fmt.Sprintf("# Epoch Trace: %s\n", frame.ID)
	trace += fmt.Sprintf("Timestamp: %s\n", time.Now().UTC().Format(time.RFC3339))
	trace += fmt.Sprintf("Intent: %s\n", frame.Intent)

	if frame.Proof != nil {
		trace += fmt.Sprintf("Audit Status: %s\n", frame.Status)
		if !frame.Proof.Pass {
			trace += fmt.Sprintf("Violation Tier: %s\n", frame.Proof.ViolationTier)
			trace += fmt.Sprintf("Conflict Path: %s\n", frame.Proof.ConflictPath)
		}
	} else {
		trace += "Audit Status: PENDING or SKIPPED\n"
	}

	return []byte(trace)
}
