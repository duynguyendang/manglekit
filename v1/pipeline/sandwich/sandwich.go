package sandwich

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/duynguyendang/manglekit/v1/core"
	"github.com/duynguyendang/manglekit/v1/internal/statehelper"
	"github.com/duynguyendang/manglekit/v1/pipeline"
	"github.com/google/uuid"
)

// Orchestrator implements the default MangleKit orchestrator. It has been refactored
// to use a typed, stage-based pipeline architecture, where a `Runner` executes
// a sequence of `Stage` implementations. This approach promotes separation of
// concerns, testability, and maintainability over the previous monolithic design.
//
// The standard flow remains:
// 1.  **Pre-retrieval rules**: Validate, normalize, and scope the user query.
// 2.  **Retrieve & Rerank**: Fetch relevant documents and refine their order.
// 3.  **LLM Call**: Synthesize an answer based on the evidence.
// 4.  **Post-retrieval rules**: Filter the final answer and citations for compliance.
type Orchestrator struct {
	action              core.Action
	subActions          map[string]core.Action
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

// Execute executes the full processing pipeline for a given query. It constructs
// a `PipelineContext`, assembles a `Runner` with the necessary stages, and
// executes them in sequence.
func (s *Orchestrator) Execute(ctx context.Context, sessionID string, q core.Query) (core.Answer, error) {
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
	runner.Add(&ActionStage{DefaultAction: s.action, SubActions: s.subActions, Logger: logger, Meter: s.obs.Meter})
	runner.Add(&RerankStage{Reranker: s.reranker, TopK: s.topK, FallbackThreshold: s.fallbackThreshold, Logger: logger, Meter: s.obs.Meter})
	runner.Add(&LLMStage{LLM: s.llm, MaxTokens: s.maxTokens, Logger: logger, Meter: s.obs.Meter})
	runner.Add(&PostRulesStage{RuleSet: s.ruleset, Logger: logger, Meter: s.obs.Meter})

	// 4. Run the pipeline.
	if err := runner.Run(p); err != nil {
		logger.Errorf("pipeline run failed", "error", err)
		// Return the partially populated answer object for inspection.
		return p.Answer, fmt.Errorf("sandwich pipeline failed: %w", err)
	}

	// 5. Update and save conversation state on a successful run.
	s.conversationManager.UpdateAndSaveHistory(ctx, sessionID, s.stateProvider, logger, p.History, p.Query, p.Answer)

	logger.Infof("pipeline run finished successfully")
	return p.Answer, nil
}

// Close releases any external resources held by the orchestrator's components.
func (s *Orchestrator) Close(ctx context.Context) error {
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
