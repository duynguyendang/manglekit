package sandwich

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/duynguyendang/manglekit/core"
	obslogger "github.com/duynguyendang/manglekit/internal/logger"
	"github.com/duynguyendang/manglekit/internal/statehelper"
	"github.com/duynguyendang/manglekit/pipeline"
	"github.com/google/uuid"
)

// SandwichOptions defines the configuration for the Sandwich orchestrator.
type SandwichOptions struct {
	Retriever         string  `yaml:"retriever"`
	Reranker          string  `yaml:"reranker,omitempty"` // Optional
	LLM               string  `yaml:"llm"`
	RuleSet           string  `yaml:"ruleSet,omitempty"`
	StateProvider     string  `yaml:"stateProvider,omitempty"`
	TopK              int     `yaml:"top_k,omitempty"`
	MaxTokens         int     `yaml:"max_tokens,omitempty"`
	FallbackThreshold float64 `yaml:"fallback_threshold,omitempty"`
}

func (o *SandwichOptions) ProviderName() string { return "sandwich" }
func (o *SandwichOptions) ProviderKind() core.Kind { return core.KindOrchestrator }

// Sandwich implements the default MangleKit orchestrator. It has been refactored
// to use a typed, stage-based pipeline architecture, where a `Runner` executes
// a sequence of `Stage` implementations. This approach promotes separation of
// concerns, testability, and maintainability over the previous monolithic design.
//
// The standard flow remains:
// 1.  **Pre-retrieval rules**: Validate, normalize, and scope the user query.
// 2.  **Retrieve & Rerank**: Fetch relevant documents and refine their order.
// 3.  **LLM Call**: Synthesize an answer based on the evidence.
// 4.  **Post-retrieval rules**: Filter the final answer and citations for compliance.
type Sandwich struct {
	retriever           core.Retriever
	reranker            core.Reranker
	ruleset             core.RuleSet
	llm                 core.LLMClient
	stateProvider       core.StateProvider
	closers             []core.ResourceCloser
	conversationManager *statehelper.ConversationManager
	obs                 core.Observability
	topK                int
	maxTokens           int
	fallbackThreshold   float64
}

// NewSandwich is the factory for the Sandwich orchestrator. It now receives a
// `core.Resolved` struct, which contains all its dependencies, and a
// `SandwichOptions` struct to explicitly configure which components to use.
// This eliminates non-deterministic behavior from map iteration.
func NewSandwich(ctx context.Context, deps core.Resolved, cfg *SandwichOptions) (core.Orchestrator, error) {
	s := &Sandwich{
		conversationManager: statehelper.NewConversationManager(),
		closers:             deps.Closers,
		obs:                 deps.Obs,
		topK:                cfg.TopK,
		maxTokens:           cfg.MaxTokens,
		fallbackThreshold:   cfg.FallbackThreshold,
	}

	// Explicitly look up components based on configuration.
	var ok bool
	if s.retriever, ok = deps.Retrievers[cfg.Retriever]; !ok {
		return nil, fmt.Errorf("retriever %q not found", cfg.Retriever)
	}
	if s.llm, ok = deps.LLMs[cfg.LLM]; !ok {
		return nil, fmt.Errorf("llm %q not found", cfg.LLM)
	}
	if cfg.Reranker != "" {
		if s.reranker, ok = deps.Rerankers[cfg.Reranker]; !ok {
			return nil, fmt.Errorf("reranker %q not found", cfg.Reranker)
		}
	}

	if cfg.RuleSet != "" {
		if s.ruleset, ok = deps.Rules[cfg.RuleSet]; !ok {
			return nil, fmt.Errorf("ruleset %q not found", cfg.RuleSet)
		}
	}
	if cfg.StateProvider != "" {
		if s.stateProvider, ok = deps.StateProviders[cfg.StateProvider]; !ok {
			return nil, fmt.Errorf("state provider %q not found", cfg.StateProvider)
		}
	}

	if s.obs.Logger == nil {
		s.obs.Logger = obslogger.NewStdLogger()
	}
	return s, nil
}

// Execute executes the full processing pipeline for a given query. It constructs
// a `PipelineContext`, assembles a `Runner` with the necessary stages, and
// executes them in sequence.
func (s *Sandwich) Execute(ctx context.Context, sessionID string, q core.Query) (core.Answer, error) {
	requestID := uuid.NewString()
	logger := s.obs.Logger.With("request_id", requestID, "pipeline", "sandwich", "session_id", sessionID)
	logger.Infof("pipeline run started", "query", q.Text)

	if s.obs.Tracer != nil {
		endTrace := s.obs.Tracer.StartSpan("manglekit.Execute")
		defer endTrace()
	}

	// 1. Initialize the PipelineContext, the typed data carrier for the run.
	p := &pipeline.PipelineContext{
		Ctx:       ctx,
		Query:     q,
		SessionID: sessionID,
		StartTime: time.Now(),
		Answer: core.Answer{
			Meta: make(map[string]any),
		},
	}

	// 2. Load conversation history if a state provider is configured.
	p.History = s.conversationManager.LoadHistory(ctx, sessionID, s.stateProvider, logger)

	// 3. Assemble the pipeline runner with stages.
	runner := &pipeline.Runner{}
	runner.Add(&PreRulesStage{RuleSet: s.ruleset, Logger: logger, Meter: s.obs.Meter})
	runner.Add(&RetrieveStage{Retriever: s.retriever, TopK: s.topK, Logger: logger, Meter: s.obs.Meter})
	runner.Add(&RerankStage{Reranker: s.reranker, TopK: s.topK, FallbackThreshold: s.fallbackThreshold, Logger: logger, Meter: s.obs.Meter})
	runner.Add(&LLMStage{LLM: s.llm, MaxTokens: s.maxTokens, Logger: logger, Meter: s.obs.Meter})
	runner.Add(&PostRulesStage{RuleSet: s.ruleset, Logger: logger, Meter: s.obs.Meter})

	// 4. Run the pipeline.
	if err := runner.Run(p); err != nil {
		logger.Errorf("pipeline run failed", "error", err)
		// Return the partially populated answer object for inspection.
		return p.Answer, err
	}

	// 5. Update and save conversation state on a successful run.
	s.conversationManager.UpdateAndSaveHistory(ctx, sessionID, s.stateProvider, logger, p.History, p.Query, p.Answer)

	logger.Infof("pipeline run finished successfully")
	return p.Answer, nil
}

// Close releases any external resources held by the orchestrator's components.
func (s *Sandwich) Close(ctx context.Context) error {
	var errs []error
	for _, closer := range s.closers {
		if err := closer(ctx); err != nil {
			s.obs.Logger.Warnf("error during resource cleanup: %v", err)
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("errors occurred during resource cleanup: %w", errors.Join(errs...))
	}
	return nil
}
