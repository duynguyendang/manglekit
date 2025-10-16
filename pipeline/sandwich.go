package pipeline

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/duynguyendang/manglekit/core"
	obslogger "github.com/duynguyendang/manglekit/internal/logger"
	"github.com/duynguyendang/manglekit/internal/statehelper"
	"github.com/duynguyendang/manglekit/llm"
	"github.com/duynguyendang/manglekit/rerank"
	"github.com/duynguyendang/manglekit/retrieve"
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
	opts                core.Options
	retriever           retrieve.Retriever
	reranker            rerank.Reranker
	ruleset             core.RuleSet
	llm                 llm.Client
	stateProvider       core.StateProvider
	closers             []core.ResourceCloser
	conversationManager *statehelper.ConversationManager
}

// NewSandwich creates a new Sandwich orchestrator from a set of options.
// It performs critical type assertions to ensure that the components provided
// in the `core.Options` struct (which uses `any` for flexibility) match the
// interfaces expected by the pipeline.
func NewSandwich(o core.Options) (core.Orchestrator, error) {
	s := &Sandwich{
		opts:                o,
		closers:             o.ResourceClosers,
		conversationManager: statehelper.NewConversationManager(),
	}
	if s.opts.Obs.Logger == nil {
		s.opts.Obs.Logger = obslogger.NewStdLogger()
	}
	var ok bool

	if o.Retriever != nil {
		s.retriever, ok = o.Retriever.(retrieve.Retriever)
		if !ok {
			return nil, fmt.Errorf("invalid retriever type: %T", o.Retriever)
		}
	}

	if o.LLM != nil {
		s.llm, ok = o.LLM.(llm.Client)
		if !ok {
			return nil, fmt.Errorf("invalid llm type: %T", o.LLM)
		}
	}

	if o.Reranker != nil {
		s.reranker, ok = o.Reranker.(rerank.Reranker)
		if !ok {
			return nil, fmt.Errorf("invalid reranker type: %T", o.Reranker)
		}
	}

	if o.StateProvider != nil {
		s.stateProvider, ok = o.StateProvider.(core.StateProvider)
		if !ok {
			return nil, fmt.Errorf("invalid state provider type: %T", o.StateProvider)
		}
	}

	if o.Rules != nil {
		s.ruleset, ok = o.Rules.(core.RuleSet)
		if !ok {
			return nil, fmt.Errorf("invalid ruleset type: %T", o.Rules)
		}
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
	var combined error
	for i := len(s.closers) - 1; i >= 0; i-- {
		if s.closers[i] == nil {
			continue
		}
		if err := s.closers[i](ctx); err != nil {
			combined = errors.Join(combined, err)
		}
	}
	return combined
}
