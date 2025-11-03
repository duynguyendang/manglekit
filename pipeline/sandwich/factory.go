package sandwich

import (
	"context"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
	"github.com/duynguyendang/manglekit/internal/statehelper"
	"github.com/duynguyendang/manglekit/internal/logger"
)

// Factory implements the core.Factory interface for the sandwich orchestrator.
type Factory struct {
	cfg *Options
}

// NewFactory creates a new factory for the sandwich orchestrator.
func NewFactory(cfg *Options) *Factory {
	return &Factory{cfg: cfg}
}

// Build creates a new sandwich orchestrator from the given dependencies.
func (f *Factory) Build(ctx context.Context, deps diapi.SandwichDeps) (core.Orchestrator, error) {
	s := &Orchestrator{
		retriever:           deps.Retriever,
		reranker:            deps.Reranker,
		ruleset:             deps.RuleSet,
		llm:                 deps.LLM,
		stateProvider:       deps.StateProvider,
		conversationManager: statehelper.NewConversationManager(),
		obs:                 deps.Obs,
		topK:                f.cfg.TopK,
		maxTokens:           f.cfg.MaxTokens,
		fallbackThreshold:   f.cfg.FallbackThreshold,
	}

	if s.obs.Logger == nil {
		s.obs.Logger = logger.NewStdLogger()
	}

	return s, nil
}
