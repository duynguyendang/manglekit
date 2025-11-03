package sandwich

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
)

// sandwichHandler implements the ComponentHandler for the sandwich orchestrator.
type sandwichHandler struct{}

// NewHandler returns a new ComponentHandler for the sandwich orchestrator.
func NewHandler() core.ComponentHandler {
	return &sandwichHandler{}
}

// Kind returns the kind of component this handler builds.
func (h *sandwichHandler) Kind() core.Kind {
	return core.KindOrchestrator
}

// BuildComponent constructs a sandwich orchestrator.
func (h *sandwichHandler) BuildComponent(
	ctx context.Context,
	builderDI any,
	factory any,
	resolved *core.Resolved,
	cfg core.ProviderOptions,
	name string,
) (core.ResourceCloser, error) {
	builder, ok := builderDI.(diapi.Builder)
	if !ok {
		return nil, fmt.Errorf("invalid builderDI type: %T", builderDI)
	}
	opts, ok := cfg.(*Options)
	if !ok {
		return nil, fmt.Errorf("invalid options type for sandwich orchestrator, got %T", cfg)
	}

	// The handler is responsible for resolving all dependencies.
	deps, err := h.buildDeps(builder, opts)
	if err != nil {
		return nil, err
	}

	// Now, build the orchestrator.
	f, ok := factory.(*Factory)
	if !ok {
		return nil, fmt.Errorf("invalid factory type for sandwich orchestrator, got %T", factory)
	}
	orch, err := f.Build(ctx, deps)
	if err != nil {
		return nil, fmt.Errorf("failed to build sandwich orchestrator: %w", err)
	}

	resolved.Orchestrators[name] = orch
	return orch.Close, nil
}

func (h *sandwichHandler) buildDeps(builder diapi.Builder, opts *Options) (diapi.SandwichDeps, error) {
	deps := diapi.SandwichDeps{
		CoreDeps: builder.GetCoreDeps(),
	}
	var err error

	deps.Retriever, err = builder.GetRetriever(opts.Retriever)
	if err != nil {
		return diapi.SandwichDeps{}, fmt.Errorf("sandwich orchestrator: failed to get retriever %q: %w", opts.Retriever, err)
	}

	deps.LLM, err = builder.GetLLMClient(opts.LLM)
	if err != nil {
		return diapi.SandwichDeps{}, fmt.Errorf("sandwich orchestrator: failed to get llm %q: %w", opts.LLM, err)
	}

	if opts.Reranker != "" {
		deps.Reranker, err = builder.GetReranker(opts.Reranker)
		if err != nil {
			return diapi.SandwichDeps{}, fmt.Errorf("sandwich orchestrator: failed to get reranker %q: %w", opts.Reranker, err)
		}
	}

	if opts.StateProvider != "" {
		deps.StateProvider, err = builder.GetStateProvider(opts.StateProvider)
		if err != nil {
			return diapi.SandwichDeps{}, fmt.Errorf("sandwich orchestrator: failed to get state provider %q: %w", opts.StateProvider, err)
		}
	}

	if opts.RuleSet != "" {
		deps.RuleSet, err = builder.GetRuleSet(opts.RuleSet)
		if err != nil {
			return diapi.SandwichDeps{}, fmt.Errorf("sandwich orchestrator: failed to get rule set %q: %w", opts.RuleSet, err)
		}
	}
	return deps, nil
}
