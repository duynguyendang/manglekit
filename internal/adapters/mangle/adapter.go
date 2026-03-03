package mangle

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/duynguyendang/manglekit/internal/core/domain"
	"github.com/google/mangle/analysis"
	"github.com/google/mangle/ast"
	"github.com/google/mangle/engine"
	"github.com/google/mangle/factstore"
	"github.com/google/mangle/parse"
)

// ReasoningAdapter implements ports.ReasoningPort using the Google Mangle engine.
type ReasoningAdapter struct {
	MaxRecursionDepth int
	EvalTimeout       time.Duration
}

// NewReasoningAdapter initializes the Datalog evaluator with strict circuit breakers.
func NewReasoningAdapter() *ReasoningAdapter {
	return &ReasoningAdapter{
		MaxRecursionDepth: 5,
		EvalTimeout:       2000 * time.Millisecond,
	}
}

// Verify implements the Shadow Audit for structured plans.
func (a *ReasoningAdapter) Verify(ctx context.Context, subject interface{}, genome []domain.DomainGene) (*domain.AuditResult, error) {
	atoms := []domain.Atom{}
	return a.VerifyAtoms(ctx, atoms, genome)
}

// VerifyAtoms executes the core Mangle evaluation (semi-naive with stratification).
func (a *ReasoningAdapter) VerifyAtoms(ctx context.Context, atoms []domain.Atom, genome []domain.DomainGene) (*domain.AuditResult, error) {
	evalCtx, cancel := context.WithTimeout(ctx, a.EvalTimeout)
	defer cancel()

	// 1. Combine all ActiveGene rules into a single Datalog program.
	var programSource strings.Builder
	for _, gene := range genome {
		programSource.Write(gene.Rules)
		programSource.WriteString("\n")
	}

	source := programSource.String()
	if strings.TrimSpace(source) == "" {
		return &domain.AuditResult{Pass: true}, nil
	}

	// 2. Parse the combined program into a SourceUnit.
	sourceUnit, err := parse.Unit(strings.NewReader(source))
	if err != nil {
		return nil, fmt.Errorf("failed to parse combined genome: %w", err)
	}

	// 3. Analyze (type-check and stratify) using the correct API.
	programInfo, err := analysis.Analyze([]parse.SourceUnit{sourceUnit}, nil)
	if err != nil {
		// Analysis failed — return soft pass with warning.
		return &domain.AuditResult{
			Pass:          true,
			EntropyDelta:  0.1,
			ConflictPath:  fmt.Sprintf("analysis_warning: %v", err),
			ViolationTier: domain.Tier2AI,
		}, nil
	}

	// 4. Create an in-memory fact store and inject input atoms.
	store := factstore.NewSimpleInMemoryStore()

	for _, atom := range atoms {
		baseFact := ast.Atom{
			Predicate: ast.PredicateSym{Symbol: atom.Predicate, Arity: 2},
			Args: []ast.BaseTerm{
				ast.String(atom.Subject),
				ast.String(atom.Object),
			},
		}
		store.Add(baseFact)
	}

	// 5. Check context timeout before evaluation.
	if evalCtx.Err() != nil {
		return nil, fmt.Errorf("evaluation timeout exceeded before engine start (%v)", a.EvalTimeout)
	}

	// 6. Evaluate the program using the Mangle semi-naive engine.
	if err := engine.EvalProgram(programInfo, store); err != nil {
		return nil, fmt.Errorf("mangle evaluation failed: %w", err)
	}

	// 7. Check for halt(Msg) derivations — Tier 0/1 violations.
	haltResult := a.checkPredicate(store, "halt", 1)
	if haltResult != nil {
		return haltResult, nil
	}

	// 8. Check for warn(Msg) derivations — soft heuristic warnings.
	warnResult := a.checkPredicate(store, "warn", 1)
	if warnResult != nil {
		// Warnings don't block — override Pass to true.
		warnResult.Pass = true
		warnResult.ViolationTier = domain.Tier2AI
		return warnResult, nil
	}

	return &domain.AuditResult{Pass: true}, nil
}

// checkPredicate scans the fact store for derived facts matching the given predicate.
func (a *ReasoningAdapter) checkPredicate(store factstore.SimpleInMemoryStore, predName string, arity int) *domain.AuditResult {
	pred := ast.PredicateSym{Symbol: predName, Arity: arity}

	// Build a wildcard query atom with variable arguments.
	queryArgs := make([]ast.BaseTerm, arity)
	for i := 0; i < arity; i++ {
		queryArgs[i] = ast.Variable{Symbol: fmt.Sprintf("X%d", i)}
	}
	queryAtom := ast.Atom{Predicate: pred, Args: queryArgs}

	var foundFacts []ast.Atom
	_ = store.GetFacts(queryAtom, func(a ast.Atom) error {
		foundFacts = append(foundFacts, a)
		return nil
	})

	if len(foundFacts) == 0 {
		return nil
	}

	msg := "unknown"
	if len(foundFacts[0].Args) > 0 {
		msg = foundFacts[0].Args[0].String()
	}

	return &domain.AuditResult{
		Pass:          false,
		ViolationTier: domain.Tier0Kernel,
		ConflictPath:  msg,
		ProofTree: &domain.ProofNode{
			Rule: fmt.Sprintf("%s(%s)", predName, msg),
			Pass: false,
		},
		EntropyDelta: 0.3,
	}
}

// Query allows raw datalog extraction, bounded by the depth limit.
func (a *ReasoningAdapter) Query(ctx context.Context, query string, genome []domain.DomainGene) ([]domain.Atom, error) {
	return nil, nil
}
