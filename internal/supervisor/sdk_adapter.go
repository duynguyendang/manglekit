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
	// Fail-closed: unknown types are treated as a Tier-0 violation.
	atoms, ok := subject.([]domain.Atom)
	if !ok {
		return &domain.AuditResult{
			Pass:          false,
			ViolationTier: domain.Tier0Kernel,
			ConflictPath:  "sdk_adapter.unknown_subject_type",
		}, nil
	}
	return a.VerifyAtoms(ctx, atoms, genome)
}

func (a *sdkEvaluatorAdapter) VerifyAtoms(ctx context.Context, atoms []domain.Atom, genome []domain.DomainGene) (*domain.AuditResult, error) {
	if len(atoms) == 0 {
		return &domain.AuditResult{Pass: true}, nil
	}

	facts := make([]string, 0, len(atoms))
	for _, atm := range atoms {
		if atm.Subject != "" && atm.Predicate != "" {
			facts = append(facts, fmt.Sprintf(`%s("%s", "%s").`, atm.Predicate, atm.Subject, atm.Object))
		}
	}

	env := core.Envelope{
		Facts:    facts,
		Metadata: make(map[string]any),
	}

	decision, err := a.inner.AssessPlan(ctx, env)

	if err != nil {
		return &domain.AuditResult{
			Pass:          false,
			ViolationTier: domain.Tier1Admin,
			ConflictPath:  "sdk_adapter.verify",
			Trail:         decision.AuditTrail,
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
			Trail:         decision.AuditTrail,
		}, nil
	}

	return &domain.AuditResult{Pass: true, Trail: decision.AuditTrail}, nil
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
		coreEnv.Facts[i] = fmt.Sprintf(`%s("%s", "%s").`, q.Predicate, q.Subject, q.Object)
	}

	result, err := a.inner.Execute(ctx, coreEnv)
	if err != nil {
		return core.Envelope{}, err
	}

	return result, nil
}

type supervisedActionV2 struct {
	inner       *SupervisedAction
	wrapped     core.Action
	extractor   Extractor // optional: converts free-text payloads to structured data
	failureMode string    // "open" or "closed" — controls extraction failure behavior
	logger      core.Logger
}

// Extractor defines the interface for converting free-text payloads to structured data.
// When set on a supervised action, it bridges the neuro-symbolic gap for unstructured
// LLM responses that would otherwise produce zero facts in flattenToQuads.
type Extractor interface {
	// Extract converts free-text input to a structured payload.
	Extract(ctx context.Context, text string) (any, error)
}

func (a *supervisedActionV2) Execute(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	payload := input.Payload

	// Opt-in extraction: if the payload is free text and an extractor is
	// registered, convert to structured data before the supervision path.
	if a.extractor != nil {
		if text, ok := payload.(string); ok {
			extracted, err := a.extractor.Extract(ctx, text)
			if err != nil {
				if a.failureMode == "open" {
					// Fail-open: log and continue with raw text
					if a.logger != nil {
						a.logger.Warn("extraction failed, continuing with raw text", "error", err)
					}
				} else {
					// Fail-closed (default): block execution
					return core.Envelope{}, fmt.Errorf("extraction failed (fail-closed): %w", err)
				}
			} else {
				payload = extracted
			}
		}
	}

	result, err := a.inner.ExecuteInternal(ctx, "default", payload)
	if err != nil {
		return core.Envelope{}, err
	}

	return result, nil
}

func (a *supervisedActionV2) Metadata() core.ActionMetadata {
	return a.wrapped.Metadata()
}

func NewSupervisedActionFromSDK(action core.Action, evaluator core.Evaluator, failureMode string, logger core.Logger) core.Action {
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
		inner:       supervised,
		wrapped:     action,
		failureMode: failureMode,
		logger:      logger,
	}
}

// WithExtractor sets the optional text-to-struct extractor on a supervised action.
// This enables the neuro-symbolic bridge: free-text LLM responses are extracted
// to structured data before flattenToQuads, so Datalog rules can fire on the result.
//
// Parameters:
//   - action: A supervised action (must be *supervisedActionV2).
//   - ext: The extractor to use for free-text payloads.
//
// Returns:
//   - The action with extractor set, or the original action if type doesn't match.
func WithExtractor(action core.Action, ext Extractor) core.Action {
	if sv, ok := action.(*supervisedActionV2); ok {
		sv.extractor = ext
	}
	return action
}
