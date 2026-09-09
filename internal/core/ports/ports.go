package ports

import (
	"context"

	"github.com/duynguyendang/manglekit/internal/core/domain"
)

// ----------------------------------------------------------------------------
// Reasoning Port (Mangle)
// ----------------------------------------------------------------------------

// ReasoningPort executes formal logic verification (Datalog evaluation).
//
// NOTE: the former "GenePool" abstraction (DomainGene, GenePoolPort, the
// genome parameter on every verify call) was removed in v0.9: no engine path
// ever consumed it. Runtime adaptation lives in ooda.Memory; learning that
// affects enforcement goes through policy sources (mkit gen -> review ->
// ReloadPolicy). See docs/adr/003-gene-pool-removed-from-kernel.md.
type ReasoningPort interface {
	// Verify checks the subject (Plan or Content) against the loaded policy.
	Verify(ctx context.Context, subject interface{}) (*domain.AuditResult, error)

	// VerifyAtoms checks a raw set of atoms against the loaded policy.
	VerifyAtoms(ctx context.Context, atoms []domain.Atom) (*domain.AuditResult, error)

	// Query executes a raw Datalog query against the loaded policy.
	Query(ctx context.Context, query string) ([]domain.Atom, error)
}
