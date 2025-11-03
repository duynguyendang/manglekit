package sandwich

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
	"github.com/duynguyendang/manglekit/internal/statehelper"
	"github.com/duynguyendang/manglekit/internal/logger"
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

	retriever, err := builder.GetRetriever(opts.Retriever)
	if err != nil {
		return nil, fmt.Errorf("sandwich orchestrator: failed to get retriever %q: %w", opts.Retriever, err)
	}

	llm, err := builder.GetLLMClient(opts.LLM)
	if err != nil {
		return nil, fmt.Errorf("sandwich orchestrator: failed to get llm %q: %w", opts.LLM, err)
	}

	var reranker core.Reranker
	if opts.Reranker != "" {
		reranker, err = builder.GetReranker(opts.Reranker)
		if err != nil {
			return nil, fmt.Errorf("sandwich orchestrator: failed to get reranker %q: %w", opts.Reranker, err)
		}
	}

	var stateProvider core.StateProvider
	if opts.StateProvider != "" {
		stateProvider, err = builder.GetStateProvider(opts.StateProvider)
		if err != nil {
			return nil, fmt.Errorf("sandwich orchestrator: failed to get state provider %q: %w", opts.StateProvider, err)
		}
	}

	var ruleSet core.RuleSet
	if opts.RuleSet != "" {
		ruleSet, err = builder.GetRuleSet(opts.RuleSet)
		if err != nil {
			return nil, fmt.Errorf("sandwich orchestrator: failed to get rule set %q: %w", opts.RuleSet, err)
		}
	}

	// Now, build the orchestrator.
	s := &Orchestrator{
		retriever:           retriever,
		reranker:            reranker,
		ruleset:             ruleSet,
		llm:                 llm,
		stateProvider:       stateProvider,
		conversationManager: statehelper.NewConversationManager(),
		closers:             resolved.Closers,
		obs:                 resolved.Obs,
		topK:                opts.TopK,
		maxTokens:           opts.MaxTokens,
		fallbackThreshold:   opts.FallbackThreshold,
	}

	if s.obs.Logger == nil {
		s.obs.Logger = logger.NewStdLogger()
	}

	resolved.Orchestrators[name] = s
	return s.Close, nil
}
