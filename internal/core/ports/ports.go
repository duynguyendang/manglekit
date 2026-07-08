package ports

import (
	"context"
	"iter"

	"github.com/duynguyendang/manglekit/internal/core/domain"
)

// ----------------------------------------------------------------------------
// 1. Reasoning Port (Mangle)
// ----------------------------------------------------------------------------

// ReasoningPort executes formal logic verification (Datalog evaluation).
type ReasoningPort interface {
	// Verify checks the subject (Plan or Content) against the provided genome (axioms).
	Verify(ctx context.Context, subject interface{}, genome []domain.DomainGene) (*domain.AuditResult, error)

	// VerifyAtoms checks a raw set of atoms against the genome.
	VerifyAtoms(ctx context.Context, atoms []domain.Atom, genome []domain.DomainGene) (*domain.AuditResult, error)

	// Query executes a raw Datalog query against the provided genome.
	Query(ctx context.Context, query string, genome []domain.DomainGene) ([]domain.Atom, error)
}

// ----------------------------------------------------------------------------
// 2. GenePool Port
// ----------------------------------------------------------------------------

// GenePoolPort handles hot-reloading and querying of the tiered knowledge base.
type GenePoolPort interface {
	// ActiveGenes returns the iterator of genes available for the current context.
	ActiveGenes(ctx context.Context, intent domain.IntentStr) iter.Seq[*domain.DomainGene]

	// Reload refreshes the gene pool from the underlying storage.
	Reload(ctx context.Context) error
}
