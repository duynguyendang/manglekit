package supervisor

import (
	"context"
	"fmt"
	"iter"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/core/domain"
	"github.com/duynguyendang/manglekit/internal/core/ports"
)

type sdkEvaluatorAdapter struct {
	inner core.Evaluator
}

func (a *sdkEvaluatorAdapter) Verify(ctx context.Context, subject interface{}, genome []domain.DomainGene) (*domain.AuditResult, error) {
	atoms, ok := subject.([]domain.Atom)
	if !ok {
		return &domain.AuditResult{Pass: true}, nil
	}
	return a.VerifyAtoms(ctx, atoms, genome)
}

func (a *sdkEvaluatorAdapter) VerifyAtoms(ctx context.Context, atoms []domain.Atom, genome []domain.DomainGene) (*domain.AuditResult, error) {
	if len(atoms) == 0 {
		return &domain.AuditResult{Pass: true}, nil
	}

	env := core.Envelope{
		Payload: map[string]any{
			"atoms": atoms,
		},
		Metadata: make(map[string]any),
	}

	decision, err := a.inner.AssessPlan(ctx, env)
	if err != nil {
		return &domain.AuditResult{
			Pass:          false,
			ViolationTier: domain.Tier1Admin,
			ConflictPath:  "sdk_adapter.verify",
		}, nil
	}

	if decision.Outcome == core.DecisionHalt {
		conflictPath := "unknown"
		if len(decision.Reasons) > 0 {
			conflictPath = decision.Reasons[0]
		}
		return &domain.AuditResult{
			Pass:          false,
			ViolationTier: domain.Tier1Admin,
			ConflictPath:  conflictPath,
		}, nil
	}

	return &domain.AuditResult{Pass: true}, nil
}

func (a *sdkEvaluatorAdapter) Query(ctx context.Context, query string, genome []domain.DomainGene) ([]domain.Atom, error) {
	results, err := a.inner.Query(ctx, []string{}, query)
	if err != nil {
		return nil, err
	}

	atoms := make([]domain.Atom, 0, len(results))
	for _, r := range results {
		for pred, val := range r {
			atoms = append(atoms, domain.Atom{
				Predicate: pred,
				Object:    val,
				Weight:    1.0,
			})
		}
	}
	return atoms, nil
}

type sdkGenePoolAdapter struct {
	evaluator core.Evaluator
}

func (a *sdkGenePoolAdapter) ActiveGenes(ctx context.Context, intent domain.IntentStr) iter.Seq[*domain.DomainGene] {
	return func(yield func(*domain.DomainGene) bool) {
		genes := a.loadGenesForIntent(ctx, string(intent))
		for i := range genes {
			if !yield(&genes[i]) {
				return
			}
		}
	}
}

func (a *sdkGenePoolAdapter) loadGenesForIntent(ctx context.Context, intent string) []domain.DomainGene {
	var genes []domain.DomainGene

	genes = append(genes, domain.DomainGene{
		Name:         "sdk_default",
		Tier:         domain.Tier1Admin,
		TierID:       "admin.default",
		Rules:        []byte{},
		Capabilities: []string{"audit"},
		Intents:      []string{intent},
	})

	return genes
}

func (a *sdkGenePoolAdapter) Reload(ctx context.Context) error {
	return nil
}

type sdkActionAdapter struct {
	inner    core.Action
	verifier ports.ReasoningPort
	genePool ports.GenePoolPort
}

func (a *sdkActionAdapter) Execute(ctx context.Context, input domain.Envelope) (domain.Envelope, error) {
	coreEnv := core.Envelope{
		Payload:  input.Payload,
		Metadata: input.Metadata,
		Facts:    make([]string, len(input.ContextFacts)),
	}

	for i, q := range input.ContextFacts {
		coreEnv.Facts[i] = fmt.Sprintf("%s(%s, %s).", q.Predicate, q.Subject, q.Object)
	}

	result, err := a.inner.Execute(ctx, coreEnv)
	if err != nil {
		return core.Envelope{}, err
	}

	return result, nil
}

type supervisedActionV2 struct {
	inner   *SupervisedAction
	wrapped core.Action
}

func (a *supervisedActionV2) Execute(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	result, err := a.inner.ExecuteInternal(ctx, "default", input.Payload)
	if err != nil {
		return core.Envelope{}, err
	}

	return result, nil
}

func (a *supervisedActionV2) Metadata() core.ActionMetadata {
	return a.wrapped.Metadata()
}

func NewSupervisedActionFromSDK(action core.Action, evaluator core.Evaluator) core.Action {
	reasoningAdapter := &sdkEvaluatorAdapter{inner: evaluator}
	genePoolAdapter := &sdkGenePoolAdapter{evaluator: evaluator}

	innerAdapter := &sdkActionAdapter{
		inner:    action,
		verifier: reasoningAdapter,
		genePool: genePoolAdapter,
	}

	supervised := &SupervisedAction{
		inner:    innerAdapter,
		verifier: reasoningAdapter,
		genePool: genePoolAdapter,
	}

	return &supervisedActionV2{
		inner:   supervised,
		wrapped: action,
	}
}
