package sandwich

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
)

// Handler is the component handler for the sandwich orchestrator.
type Handler struct{}

// NewHandler returns a new ComponentHandler for the sandwich orchestrator.
func NewHandler() core.ComponentHandler {
	return &Handler{}
}

// Kind returns the component kind.
func (h *Handler) Kind() core.Kind {
	return core.KindOrchestrator
}

// BuildComponent builds the sandwich orchestrator.
func (h *Handler) BuildComponent(
	ctx context.Context,
	builderDI any,
	factory any,
	resolved *core.Resolved,
	cfg core.ProviderOptions,
	name string,
) (core.ResourceCloser, error) {
	b, ok := builderDI.(diapi.Builder)
	if !ok {
		return nil, fmt.Errorf("invalid builder DI type for sandwich orchestrator: got %T", builderDI)
	}

	opts, ok := cfg.(*Options)
	if !ok {
		return nil, fmt.Errorf("invalid options type for sandwich orchestrator, got %T", cfg)
	}

	retriever, err := b.GetRetriever(opts.Retriever)
	if err != nil {
		return nil, fmt.Errorf("sandwich orchestrator: failed to get retriever %q: %w", opts.Retriever, err)
	}

	llm, err := b.GetLLMClient(opts.LLM)
	if err != nil {
		return nil, fmt.Errorf("sandwich orchestrator: failed to get llm %q: %w", opts.LLM, err)
	}

	var reranker core.Reranker
	if opts.Reranker != "" {
		reranker, err = b.GetReranker(opts.Reranker)
		if err != nil {
			return nil, fmt.Errorf("sandwich orchestrator: failed to get reranker %q: %w", opts.Reranker, err)
		}
	}

	var ruleSet core.RuleSet
	if opts.RuleSet != "" {
		ruleSet, err = b.GetRuleSet(opts.RuleSet)
		if err != nil {
			return nil, fmt.Errorf("sandwich orchestrator: failed to get ruleset %q: %w", opts.RuleSet, err)
		}
	}

	var stateProvider core.StateProvider
	if opts.StateProvider != "" {
		stateProvider, err = b.GetStateProvider(opts.StateProvider)
		if err != nil {
			return nil, fmt.Errorf("sandwich orchestrator: failed to get state provider %q: %w", opts.StateProvider, err)
		}
	}

	deps := diapi.SandwichDeps{
		CoreDeps:      b.GetCoreDeps(),
		Retriever:     retriever,
		LLM:           llm,
		Reranker:      reranker,
		RuleSet:       ruleSet,
		StateProvider: stateProvider,
	}

	f, ok := factory.(core.Factory)
	if !ok {
		return nil, fmt.Errorf("invalid factory type for sandwich orchestrator: got %T", factory)
	}

	built, err := f.Build(ctx, deps, cfg)
	if err != nil {
		return nil, fmt.Errorf("factory for %s '%s' failed: %w", core.KindOrchestrator, name, err)
	}

	orchestrator, ok := built.(core.Orchestrator)
	if !ok {
		return nil, fmt.Errorf("component %s is not a valid orchestrator", name)
	}
	resolved.Orchestrators[name] = orchestrator
	return orchestrator.Close, nil
}
