package audit

import (
	"context"
	"errors"
	"testing"

	"github.com/duynguyendang/manglekit/internal/core/domain"
)

type mockReasoningPortForAudit struct {
	verifyResult *domain.AuditResult
	verifyErr    error
}

func (m *mockReasoningPortForAudit) Verify(ctx context.Context, subject interface{}, genome []domain.DomainGene) (*domain.AuditResult, error) {
	return m.verifyResult, m.verifyErr
}

func (m *mockReasoningPortForAudit) VerifyAtoms(ctx context.Context, atoms []domain.Atom, genome []domain.DomainGene) (*domain.AuditResult, error) {
	return m.verifyResult, m.verifyErr
}

func (m *mockReasoningPortForAudit) Query(ctx context.Context, query string, genome []domain.DomainGene) ([]domain.Atom, error) {
	return nil, nil
}

func TestAuditor_Verify_MissingTier0(t *testing.T) {
	verifier := &mockReasoningPortForAudit{}
	auditor := New(verifier)

	frame := &domain.CognitiveFrame{
		ID:     [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		Intent: domain.IntentStr("test"),
		ActiveGenes: []domain.DomainGene{
			{Name: "Tier1Gene", Tier: domain.Tier1Admin},
		},
		OutputType: domain.OutputTypePlan,
	}

	_, err := auditor.Verify(context.Background(), frame)
	if err == nil {
		t.Fatal("expected error for missing Tier 0")
	}
}

func TestAuditor_Verify_PlanPass(t *testing.T) {
	verifier := &mockReasoningPortForAudit{
		verifyResult: &domain.AuditResult{Pass: true},
	}
	auditor := New(verifier)

	frame := &domain.CognitiveFrame{
		ID:     [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		Intent: domain.IntentStr("test"),
		ActiveGenes: []domain.DomainGene{
			{Name: "Kernel", Tier: domain.Tier0Kernel},
		},
		OutputType: domain.OutputTypePlan,
		Draft:      "clean plan",
	}

	result, err := auditor.Verify(context.Background(), frame)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Pass {
		t.Errorf("expected Pass=true, got false")
	}
	if frame.Status != domain.VerifyStatusPassed {
		t.Errorf("expected Status Passed, got %v", frame.Status)
	}
}

func TestAuditor_Verify_PlanFailTier0(t *testing.T) {
	verifier := &mockReasoningPortForAudit{
		verifyResult: &domain.AuditResult{
			Pass:          false,
			ViolationTier: domain.Tier0Kernel,
		},
	}
	auditor := New(verifier)

	frame := &domain.CognitiveFrame{
		ID:     [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		Intent: domain.IntentStr("test"),
		ActiveGenes: []domain.DomainGene{
			{Name: "Kernel", Tier: domain.Tier0Kernel},
		},
		OutputType: domain.OutputTypePlan,
		Draft:      "bad plan",
	}

	result, err := auditor.Verify(context.Background(), frame)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Pass {
		t.Errorf("expected Pass=false, got true")
	}
	if frame.Status != domain.VerifyStatusFailed {
		t.Errorf("expected Status Failed, got %v", frame.Status)
	}
}

func TestAuditor_Verify_PlanFailTier1(t *testing.T) {
	verifier := &mockReasoningPortForAudit{
		verifyResult: &domain.AuditResult{
			Pass:          false,
			ViolationTier: domain.Tier1Admin,
		},
	}
	auditor := New(verifier)

	frame := &domain.CognitiveFrame{
		ID:     [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		Intent: domain.IntentStr("test"),
		ActiveGenes: []domain.DomainGene{
			{Name: "Kernel", Tier: domain.Tier0Kernel},
		},
		OutputType: domain.OutputTypePlan,
		Draft:      "bad plan",
	}

	result, err := auditor.Verify(context.Background(), frame)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Pass {
		t.Errorf("expected Pass=false, got true")
	}
	if frame.Status != domain.VerifyStatusFailed {
		t.Errorf("expected Status Failed for Tier1Admin violation, got %v", frame.Status)
	}
}

func TestAuditor_Verify_PlanFailTier2(t *testing.T) {
	verifier := &mockReasoningPortForAudit{
		verifyResult: &domain.AuditResult{
			Pass:          false,
			ViolationTier: domain.Tier2AI,
		},
	}
	auditor := New(verifier)

	frame := &domain.CognitiveFrame{
		ID:     [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		Intent: domain.IntentStr("test"),
		ActiveGenes: []domain.DomainGene{
			{Name: "Kernel", Tier: domain.Tier0Kernel},
		},
		OutputType: domain.OutputTypePlan,
		Draft:      "bad plan",
	}

	result, err := auditor.Verify(context.Background(), frame)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Pass {
		t.Errorf("expected Pass=true (overridden from false for soft violation), got false")
	}
	if frame.Status != domain.VerifyStatusWarning {
		t.Errorf("expected Status Warning for Tier2/3 violations, got %v", frame.Status)
	}
}

func TestAuditor_Verify_UnknownOutputType(t *testing.T) {
	verifier := &mockReasoningPortForAudit{}
	auditor := New(verifier)

	frame := &domain.CognitiveFrame{
		ID:     [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		Intent: domain.IntentStr("test"),
		ActiveGenes: []domain.DomainGene{
			{Name: "Kernel", Tier: domain.Tier0Kernel},
		},
		OutputType: domain.OutputType("UNKNOWN"),
	}

	_, err := auditor.Verify(context.Background(), frame)
	if err == nil {
		t.Fatal("expected error for unknown output type")
	}
}

func TestAuditor_VerifyInducedRules(t *testing.T) {
	verifier := &mockReasoningPortForAudit{}
	auditor := New(verifier)

	frame := &domain.CognitiveFrame{
		ID:     [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		Intent: domain.IntentStr("test"),
		ActiveGenes: []domain.DomainGene{
			{Name: "Kernel", Tier: domain.Tier0Kernel},
		},
		OutputType: domain.OutputTypeRule,
		Draft:      []byte("rule(a,b)."),
	}

	result, err := auditor.Verify(context.Background(), frame)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Pass {
		t.Errorf("expected Pass=true for valid induced rules, got false")
	}
}

func TestAuditor_VerifyInducedRules_InvalidDraft(t *testing.T) {
	verifier := &mockReasoningPortForAudit{}
	auditor := New(verifier)

	frame := &domain.CognitiveFrame{
		ID:     [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		Intent: domain.IntentStr("test"),
		ActiveGenes: []domain.DomainGene{
			{Name: "Kernel", Tier: domain.Tier0Kernel},
		},
		OutputType: domain.OutputTypeRule,
		Draft:      "not bytes",
	}

	_, err := auditor.Verify(context.Background(), frame)
	if err == nil {
		t.Fatal("expected error for non-bytes draft")
	}
}

func TestAuditor_GenerateTrace(t *testing.T) {
	verifier := &mockReasoningPortForAudit{}
	auditor := New(verifier)

	frame := &domain.CognitiveFrame{
		ID:     [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		Intent: domain.IntentStr("test-intent"),
		Status: domain.VerifyStatusPassed,
		Proof: &domain.AuditResult{
			Pass: true,
		},
	}

	trace := auditor.GenerateTrace(frame)
	if len(trace) == 0 {
		t.Error("expected non-empty trace")
	}
}

func TestAuditor_GenerateTrace_NoProof(t *testing.T) {
	verifier := &mockReasoningPortForAudit{}
	auditor := New(verifier)

	frame := &domain.CognitiveFrame{
		ID:     [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		Intent: domain.IntentStr("test-intent"),
		Status: domain.VerifyStatusPending,
	}

	trace := auditor.GenerateTrace(frame)
	if len(trace) == 0 {
		t.Error("expected non-empty trace even without proof")
	}
}

func TestAuditor_Verify_VerifierError(t *testing.T) {
	verifier := &mockReasoningPortForAudit{
		verifyErr: errors.New("verifier failed"),
	}
	auditor := New(verifier)

	frame := &domain.CognitiveFrame{
		ID:     [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		Intent: domain.IntentStr("test"),
		ActiveGenes: []domain.DomainGene{
			{Name: "Kernel", Tier: domain.Tier0Kernel},
		},
		OutputType: domain.OutputTypePlan,
		Draft:      "plan",
	}

	_, err := auditor.Verify(context.Background(), frame)
	if err == nil {
		t.Fatal("expected error from verifier")
	}
}