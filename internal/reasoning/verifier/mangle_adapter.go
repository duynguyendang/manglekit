package verifier

import (
	"context"
	"fmt"
	"strings"

	"github.com/duynguyendang/manglekit-wip/internal/core/domain"
	"github.com/duynguyendang/manglekit-wip/internal/core/ports"
)

// MangleVerifier enforces post-generation content logic rules (LLD 8.2).
// It verifies raw string output against style or structural constraints.
type MangleVerifier struct {
	base ports.ReasoningPort
}

// Verify implements content analysis against structural Gene rules.
func (v *MangleVerifier) Verify(ctx context.Context, subject interface{}, genome []domain.DomainGene) (*domain.AuditResult, error) {

	// In LLD 8.2, this checks sectioning and style constraints (like prohibited words).
	// We'll mock the check of a simple rule.

	draftStr, ok := subject.(string)
	if !ok {
		// Can only content-verify strings/raw bytes
		return v.base.Verify(ctx, subject, genome)
	}

	// Simplistic simulation of Datalog rule string matching.
	// Actual Mangle adapter would inject the document as atoms and query `prohibited(Word)`.
	prohibited := []string{"unverified_claim", "dangerous_code"}

	for _, word := range prohibited {
		if strings.Contains(strings.ToLower(draftStr), word) {
			return &domain.AuditResult{
				Pass:          false,
				ViolationTier: domain.Tier1Admin,
				ConflictPath:  "style_constraint",
				ProofTree: &domain.ProofNode{
					Rule: fmt.Sprintf("contains_prohibited_word(%s)", word),
				},
			}, nil
		}
	}

	return &domain.AuditResult{Pass: true}, nil
}

func (v *MangleVerifier) VerifyAtoms(ctx context.Context, atoms []domain.Atom, genome []domain.DomainGene) (*domain.AuditResult, error) {
	return v.base.VerifyAtoms(ctx, atoms, genome)
}
func (v *MangleVerifier) Query(ctx context.Context, query string, genome []domain.DomainGene) ([]domain.Atom, error) {
	return v.base.Query(ctx, query, genome)
}
