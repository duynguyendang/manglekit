package verifier

import (
	"context"
	"errors"
	"testing"

	"github.com/duynguyendang/manglekit/internal/core/domain"
)

type mockReasoningPort struct {
	verifyResult    *domain.AuditResult
	verifyErr       error
	verifyAtomsResult *domain.AuditResult
	verifyAtomsErr    error
}

func (m *mockReasoningPort) Verify(ctx context.Context, subject interface{}, genome []domain.DomainGene) (*domain.AuditResult, error) {
	return m.verifyResult, m.verifyErr
}

func (m *mockReasoningPort) VerifyAtoms(ctx context.Context, atoms []domain.Atom, genome []domain.DomainGene) (*domain.AuditResult, error) {
	return m.verifyAtomsResult, m.verifyAtomsErr
}

func (m *mockReasoningPort) Query(ctx context.Context, query string, genome []domain.DomainGene) ([]domain.Atom, error) {
	return nil, nil
}

func TestMangleVerifier_Verify_CleanText(t *testing.T) {
	base := &mockReasoningPort{}
	v := &MangleVerifier{base: base}

	result, err := v.Verify(context.Background(), "this is clean text", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Pass {
		t.Errorf("expected Pass=true for clean text, got false")
	}
}

func TestMangleVerifier_Verify_ProhibitedWord(t *testing.T) {
	base := &mockReasoningPort{}
	v := &MangleVerifier{base: base}

	result, err := v.Verify(context.Background(), "this contains unverified_claim in text", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Pass {
		t.Errorf("expected Pass=false for prohibited word, got true")
	}
	if result.ViolationTier != domain.Tier1Admin {
		t.Errorf("expected ViolationTier Tier1Admin, got %v", result.ViolationTier)
	}
	if result.ConflictPath != "style_constraint" {
		t.Errorf("expected ConflictPath 'style_constraint', got %q", result.ConflictPath)
	}
}

func TestMangleVerifier_Verify_NonStringDelegatesToBase(t *testing.T) {
	base := &mockReasoningPort{
		verifyResult: &domain.AuditResult{Pass: true},
	}
	v := &MangleVerifier{base: base}

	result, err := v.Verify(context.Background(), 123, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Pass {
		t.Errorf("expected Pass=true from base for non-string, got false")
	}
}

func TestMangleVerifier_Verify_DangerousCode(t *testing.T) {
	base := &mockReasoningPort{}
	v := &MangleVerifier{base: base}

	result, err := v.Verify(context.Background(), "dangerous_code snippet", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Pass {
		t.Errorf("expected Pass=false for dangerous_code, got true")
	}
	if result.ProofTree == nil {
		t.Error("expected ProofTree to be set")
	}
	if result.ProofTree == nil || result.ProofTree.Rule != "contains_prohibited_word(dangerous_code)" {
		t.Errorf("expected proof rule for dangerous_code, got %v", result.ProofTree)
	}
}

func TestMangleVerifier_VerifyAtoms_Delegates(t *testing.T) {
	base := &mockReasoningPort{
		verifyAtomsResult: &domain.AuditResult{Pass: true},
	}
	v := &MangleVerifier{base: base}

	result, err := v.VerifyAtoms(context.Background(), []domain.Atom{}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Pass {
		t.Errorf("expected Pass=true from base, got false")
	}
}

func TestMangleVerifier_Verify_EmptyString(t *testing.T) {
	base := &mockReasoningPort{}
	v := &MangleVerifier{base: base}

	result, err := v.Verify(context.Background(), "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Pass {
		t.Errorf("expected Pass=true for empty string, got false")
	}
}

func TestMangleVerifier_Verify_CaseInsensitive(t *testing.T) {
	base := &mockReasoningPort{}
	v := &MangleVerifier{base: base}

	result, err := v.Verify(context.Background(), "UNVERIFIED_CLAIM uppercase", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Pass {
		t.Errorf("expected Pass=false for uppercase prohibited word, got true")
	}
}

func TestMangleVerifier_Verify_UnknownWord(t *testing.T) {
	base := &mockReasoningPort{}
	v := &MangleVerifier{base: base}

	result, err := v.Verify(context.Background(), "this is completely clean text with no issues", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Pass {
		t.Errorf("expected Pass=true for clean text, got false")
	}
}

func TestMangleVerifier_Query_Delegates(t *testing.T) {
	base := &mockReasoningPort{}
	v := &MangleVerifier{base: base}

	_, err := v.Query(context.Background(), "test_query", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMangleVerifier_VerifyAtoms_BaseError(t *testing.T) {
	base := &mockReasoningPort{
		verifyAtomsErr: errors.New("base error"),
	}
	v := &MangleVerifier{base: base}

	_, err := v.VerifyAtoms(context.Background(), []domain.Atom{{Predicate: "test"}}, nil)
	if err == nil {
		t.Fatal("expected error from base")
	}
}