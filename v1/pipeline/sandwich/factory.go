package sandwich

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/v1/core"
	"github.com/duynguyendang/manglekit/v1/core/diapi"
	"github.com/duynguyendang/manglekit/internal/logger"
	"github.com/duynguyendang/manglekit/v1/internal/statehelper"
)

// Factory implements the core.Factory interface for the sandwich orchestrator.
type Factory struct{}

// NewFactory creates a new factory for the sandwich orchestrator.
func NewFactory() core.Factory {
	return &Factory{}
}

// Build creates a new sandwich orchestrator from the given dependencies.
func (f *Factory) Build(ctx context.Context, deps any, cfg any) (any, error) {
	sandwichDeps, ok := deps.(diapi.SandwichDeps)
	if !ok {
		return nil, fmt.Errorf("invalid deps type for sandwich orchestrator: got %T", deps)
	}

	opts, ok := cfg.(*Options)
	if !ok {
		return nil, fmt.Errorf("invalid options type for sandwich orchestrator: got %T", cfg)
	}

	s := &Orchestrator{
		action:              sandwichDeps.Action,
		subActions:          sandwichDeps.SubActions,
		reranker:            sandwichDeps.Reranker,
		ruleset:             sandwichDeps.RuleSet,
		llm:                 sandwichDeps.LLM,
		stateProvider:       sandwichDeps.StateProvider,
		conversationManager: statehelper.NewConversationManager(),
		obs:                 sandwichDeps.Obs,
		topK:                opts.TopK,
		maxTokens:           opts.MaxTokens,
		fallbackThreshold:   opts.FallbackThreshold,
	}

	if s.obs.Logger == nil {
		s.obs.Logger = logger.NewStdLogger()
	}

	return s, nil
}
