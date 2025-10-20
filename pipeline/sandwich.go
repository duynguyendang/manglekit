package pipeline

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/duynguyendang/manglekit/core"
	obslogger "github.com/duynguyendang/manglekit/internal/logger"
	"github.com/duynguyendang/manglekit/internal/statehelper"
	"github.com/google/uuid"
)

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
	opts                core.OptionsLike
	retriever           core.Retriever
	reranker            core.Reranker
	ruleset             core.RuleSet
	llm                 core.LLMClient
	stateProvider       core.StateProvider
	closers             []core.ResourceCloser
	conversationManager *statehelper.ConversationManager
}

// NewSandwich is the factory for the Sandwich orchestrator. It now receives a
// `core.Resolved` struct, which contains all its dependencies, fully constructed
// and strongly typed. This eliminates the need for runtime type assertions.
func NewSandwich(ctx context.Context, deps core.Resolved) (core.Orchestrator, error) {
	s := &Sandwich{
		retriever:           deps.Retriever,
		reranker:            deps.Reranker,
		ruleset:             deps.Rules,
		llm:                 deps.LLM,
		stateProvider:       deps.StateProvider,
		conversationManager: statehelper.NewConversationManager(),
		closers:             deps.Closers,
		opts: core.OptionsLike{ // Adapt to the new options-like struct
			TopK:              deps.TopK,
			MaxTokens:         deps.MaxTokens,
			FallbackThreshold: deps.FallbackThreshold,
			Obs:               deps.Obs,
		},
	}
	if s.opts.Obs.Logger == nil {
		s.opts.Obs.Logger = obslogger.NewStdLogger()
	}
	return s, nil
}

// Execute executes the full processing pipeline for a given query. It constructs
// a `PipelineContext`, assembles a `Runner` with the necessary stages, and
// executes them in sequence.
func (s *Sandwich) Execute(ctx context.Context, sessionID string, q core.Query) (core.Answer, error) {
	requestID := uuid.NewString()
	logger := s.opts.Obs.Logger.With("request_id", requestID, "pipeline", "sandwich", "session_id", sessionID)
	logger.Infof("pipeline run started", "query", q.Text)

	if s.opts.Obs.Tracer != nil {
		endTrace := s.opts.Obs.Tracer.StartSpan("manglekit.Execute")
		defer endTrace()
	}

	// 1. Initialize the PipelineContext, the typed data carrier for the run.
	p := &PipelineContext{
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
	runner := &Runner{}
	runner.Add(&PreRulesStage{RuleSet: s.ruleset, Logger: logger, Meter: s.opts.Obs.Meter})
	runner.Add(&RetrieveStage{Retriever: s.retriever, TopK: s.opts.TopK, Logger: logger, Meter: s.opts.Obs.Meter})
	runner.Add(&RerankStage{Reranker: s.reranker, TopK: s.opts.TopK, FallbackThreshold: s.opts.FallbackThreshold, Logger: logger, Meter: s.opts.Obs.Meter})
	runner.Add(&LLMStage{LLM: s.llm, MaxTokens: s.opts.MaxTokens, Logger: logger, Meter: s.opts.Obs.Meter})
	runner.Add(&PostRulesStage{RuleSet: s.ruleset, Logger: logger, Meter: s.opts.Obs.Meter})

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
			s.opts.Obs.Logger.Warnf("error during resource cleanup: %v", err)
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("errors occurred during resource cleanup: %w", errors.Join(errs...))
	}
	return nil
}
