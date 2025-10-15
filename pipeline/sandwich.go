package pipeline

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/duynguyendang/manglekit/core"
	obslogger "github.com/duynguyendang/manglekit/internal/logger"
	"github.com/duynguyendang/manglekit/llm"
	"github.com/duynguyendang/manglekit/rerank"
	"github.com/duynguyendang/manglekit/retrieve"
	"github.com/google/uuid"
)

// Sandwich implements the default MangleKit orchestrator, which is the most
// common execution flow. It follows a "sandwich" pattern where a central
// retrieval and generation core is "sandwiched" between two layers of rule
// evaluations:
//
// 1.  **Pre-retrieval rules**: Validate, normalize, and scope the user query.
// 2.  **Retrieve & Rerank**: Fetch relevant documents and refine their order.
// 3.  **LLM Call**: Synthesize an answer based on the evidence.
// 4.  **Post-retrieval rules**: Filter the final answer and citations for compliance.
type Sandwich struct {
	opts          core.Options
	retriever     retrieve.Retriever
	reranker      rerank.Reranker
	llm           llm.Client
	stateProvider core.StateProvider
	closers       []core.ResourceCloser
}

// NewSandwich creates a new Sandwich orchestrator from a set of options.
// It performs critical type assertions to ensure that the components provided
// in the `core.Options` struct (which uses `any` for flexibility) match the
// interfaces expected by the pipeline, such as `retrieve.Retriever` and `llm.Client`.
// This function is typically called by the framework's builder, not directly by end-users.
//
// o contains the configuration and components for the pipeline.
// It returns a configured `core.Orchestrator` or an error if any required
// component is missing or has an invalid type.
func NewSandwich(o core.Options) (core.Orchestrator, error) {
	s := &Sandwich{
		opts:    o,
		closers: o.ResourceClosers,
	}
	if s.opts.Obs.Logger == nil {
		s.opts.Obs.Logger = obslogger.NewStdLogger()
	}
	var ok bool

	if o.Retriever != nil {
		s.retriever, ok = o.Retriever.(retrieve.Retriever)
		if !ok {
			err := fmt.Errorf("invalid retriever type: %T", o.Retriever)
			s.opts.Obs.Logger.Errorf(err.Error())
			return nil, err
		}
	}

	if o.LLM != nil {
		s.llm, ok = o.LLM.(llm.Client)
		if !ok {
			err := fmt.Errorf("invalid llm type: %T", o.LLM)
			s.opts.Obs.Logger.Errorf(err.Error())
			return nil, err
		}
	}

	if o.Reranker != nil {
		s.reranker, ok = o.Reranker.(rerank.Reranker)
		if !ok {
			err := fmt.Errorf("invalid reranker type: %T", o.Reranker)
			s.opts.Obs.Logger.Errorf(err.Error())
			return nil, err
		}
	}

	if o.StateProvider != nil {
		s.stateProvider, ok = o.StateProvider.(core.StateProvider)
		if !ok {
			err := fmt.Errorf("invalid state provider type: %T", o.StateProvider)
			s.opts.Obs.Logger.Errorf(err.Error())
			return nil, err
		}
	}

	return s, nil
}

// Retriever returns the retriever component configured for the orchestrator.
// It satisfies a requirement of the `core.Orchestrator` interface, allowing access
// to the retriever instance at runtime. This is particularly useful for calling
// methods on an `Updatable` retriever to add or modify documents in a live system.
// The return type is `any` to avoid circular package dependencies; the caller is
// expected to perform a type assertion to `retrieve.Retriever` or `retrieve.Updatable`.
func (s *Sandwich) Retriever() any {
	return s.retriever
}

// Reranker returns the reranker component configured for the orchestrator.
func (s *Sandwich) Reranker() any {
	return s.reranker
}

// StateProvider returns the state provider component configured for the orchestrator.
func (s *Sandwich) StateProvider() core.StateProvider {
	return s.stateProvider
}

// Close releases any external resources held by the orchestrator (e.g., API clients).
// It iterates over the resource closers in reverse order and calls them.
//
// ctx is the context for the close operation.
// It returns a combined error if any of the closers fail.
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

// Execute executes the full processing pipeline for a given query. This typically
// involves pre-processing rules, document retrieval, reranking, LLM generation,
// and post-processing rules.
//
// ctx is the context for the entire operation.
// q is the user's query to be processed.
// It returns a final Answer containing the generated text and citations,
// or an error if any part of the process fails.
func (s *Sandwich) Execute(ctx context.Context, sessionID string, q core.Query) (core.Answer, error) {
	requestID := uuid.NewString()
	logger := s.opts.Obs.Logger.With("request_id", requestID, "pipeline", "sandwich", "session_id", sessionID)
	logger.Infof("pipeline run started", "query", q.Text)

	if s.opts.Obs.Tracer != nil {
		endTrace := s.opts.Obs.Tracer.StartSpan("manglekit.Execute")
		defer endTrace()
	}
	answer := core.Answer{
		Meta: map[string]any{},
	}

	var history core.ConversationHistory

	// 1. RETRIEVE STATE: If a state provider and sessionID are available, get the history.
	if s.stateProvider != nil && sessionID != "" {
		rawState, err := s.stateProvider.Get(ctx, sessionID)
		if err != nil {
			logger.Warnf("Failed to retrieve state for session %s: %v", sessionID, err)
			// Do not fail the request, just proceed without history.
		}

		if rawState != nil {
			// Use a type assertion or a helper function to convert the state.
			// For simplicity, we assume the state is stored as a JSON string.
			if stateBytes, ok := rawState.([]byte); ok {
				if err := json.Unmarshal(stateBytes, &history); err != nil {
					logger.Warnf("Failed to unmarshal state for session %s: %v", sessionID, err)
				}
			}
		}
	}

	// 1. Pre-Rules
	var err error
	q, err = s.runPreRules(ctx, logger, q, &answer)
	if err != nil {
		return answer, err
	}

	// 2. Retrieve
	docs, err := s.runRetrieve(ctx, logger, q, &answer)
	if err != nil {
		return answer, err
	}
	if len(docs) == 0 {
		return answer, core.ErrNoEvidence
	}

	// 3. Rerank / Fallback
	docs, err = s.runRerank(ctx, logger, q, docs, &answer)
	if err != nil {
		return answer, err
	}

	// 4. Prepare for LLM
	passages, err := s.prepareLlmRequest(docs, &answer)
	if err != nil {
		logger.Errorf("prepare llm request failed", "error", err)
		return answer, fmt.Errorf("prepare llm request failed: %w", err)
	}

	if q.Meta == nil {
		q.Meta = make(map[string]any)
	}
	q.Meta["history"] = history.Messages

	// 5. LLM
	llmErr := s.runLlm(ctx, logger, q, passages, &answer)

	// 3. UPDATE AND SAVE STATE: After a successful LLM call, update and save the history.
	if s.stateProvider != nil && sessionID != "" && llmErr == nil {
		// Append the user's query and the model's answer to the history.
		history.Messages = append(history.Messages, core.Message{Role: "user", Content: q.Text})
		history.Messages = append(history.Messages, core.Message{Role: "model", Content: answer.Text})

		// Marshal the updated history to JSON before saving.
		updatedStateBytes, err := json.Marshal(history)
		if err != nil {
			logger.Warnf("Failed to marshal updated state for session %s: %v", sessionID, err)
		} else {
			if err := s.stateProvider.Set(ctx, sessionID, updatedStateBytes); err != nil {
				logger.Warnf("Failed to save state for session %s: %v", sessionID, err)
			}
		}
	}

	if llmErr != nil {
		return answer, llmErr
	}

	// 6. Post-Rules
	if err := s.runPostRules(ctx, logger, q, &answer); err != nil {
		return answer, err
	}

	logger.Infof("pipeline run finished successfully")
	return answer, nil
}

func (s *Sandwich) runPreRules(ctx context.Context, logger core.Logger, q core.Query, a *core.Answer) (core.Query, error) {
	if s.opts.Rules == nil {
		return q, nil
	}
	meter := s.opts.Obs.Meter

	tPreRulesStart := time.Now()
	res, err := s.opts.Rules.Evaluate(core.Pre, q, nil)
	if meter != nil {
		meter.Record("manglekit.rules_pre_ms", float64(time.Since(tPreRulesStart).Milliseconds()))
	}
	if err != nil {
		logger.Errorf("pre-rules failed", "error", err)
		return q, fmt.Errorf("pre-rules failed: %w", err)
	}
	if !res.Allowed {
		if res.Mutate != nil {
			res.Mutate(&q, a)
		}
		logger.Warnf("request denied by pre-rule", "reason", res.Reason)
		return q, fmt.Errorf("%w: %s", core.ErrDenied, res.Reason)
	}
	if res.Mutate != nil {
		res.Mutate(&q, a)
		logger.Debugf("query mutated by pre-rule")
	}

	var filters any
	var expansions any
	if q.Meta != nil {
		filters = q.Meta["filters"]
		expansions = q.Meta["expansion_terms"]
	}
	logger.Debugf("pre-rules outputs", "filters", filters, "expansions", expansions)
	return q, nil
}

func (s *Sandwich) runRetrieve(ctx context.Context, logger core.Logger, q core.Query, a *core.Answer) ([]core.Doc, error) {
	meter := s.opts.Obs.Meter

	tRetrieveStart := time.Now()
	retrReq := retrieve.Request{Query: q.Text, TopK: s.opts.TopK, Meta: q.Meta}
	logger.Debugf("calling retriever", "filters", q.Meta["filters"], "expansions", q.Meta["expansion_terms"])
	retrRes, err := s.retriever.Retrieve(ctx, retrReq)
	if err != nil {
		logger.Errorf("retrieve failed", "error", err)
		return nil, fmt.Errorf("retrieve failed: %w", err)
	}
	retrieveMs := time.Since(tRetrieveStart).Milliseconds()
	a.Meta["retrieve_ms"] = retrieveMs
	if meter != nil {
		meter.Record("manglekit.retrieve_ms", float64(retrieveMs))
	}
	docs := retrRes.Docs
	a.Meta["original_docs"] = docs

	logger.Infof("retrieved documents", "count", len(docs))
	return docs, nil
}

func (s *Sandwich) runRerank(ctx context.Context, logger core.Logger, q core.Query, docs []core.Doc, a *core.Answer) ([]core.Doc, error) {
	if s.reranker == nil {
		return docs, nil
	}
	meter := s.opts.Obs.Meter

	tRerankStart := time.Now()
	rerankedDocs, err := s.reranker.Rerank(ctx, rerank.Request{Query: q.Text, Docs: docs, TopK: s.opts.TopK})
	rerankMs := time.Since(tRerankStart).Milliseconds()
	a.Meta["rerank_ms"] = rerankMs
	if meter != nil {
		meter.Record("manglekit.rerank_ms", float64(rerankMs))
	}
	if err != nil {
		logger.Errorf("rerank failed", "error", err)
		return nil, fmt.Errorf("rerank failed: %w", err)
	}
	logger.Infof("reranked documents", "count", len(rerankedDocs))
	a.Meta["reranked_docs"] = rerankedDocs

	d := make([]core.Doc, len(rerankedDocs))
	var bestScore float64
	if len(rerankedDocs) > 0 {
		bestScore = rerankedDocs[0].Score
	}
	for i, rd := range rerankedDocs {
		d[i] = rd.Doc
	}
	a.Meta["best_score"] = bestScore

	if s.opts.FallbackThreshold > 0 && bestScore < s.opts.FallbackThreshold {
		logger.Warnf("fallback threshold not met", "best_score", bestScore, "threshold", s.opts.FallbackThreshold)
		return nil, core.ErrNoEvidence
	}
	return d, nil
}

func (s *Sandwich) prepareLlmRequest(docs []core.Doc, a *core.Answer) ([]string, error) {
	passages := make([]string, len(docs))
	for i, d := range docs {
		passages[i] = d.Text
	}

	if rerankedDocs, ok := a.Meta["reranked_docs"].([]rerank.ScoredDoc); ok {
		citations := make([]core.Citation, len(rerankedDocs))
		for i, rd := range rerankedDocs {
			citations[i] = core.Citation{
				ID:      rd.Doc.ID,
				Source:  rd.Doc.Source,
				URI:     rd.Doc.URI,
				Snippet: rd.Doc.Text,
				Score:   rd.Score,
			}
		}
		a.Citations = citations
	} else {
		citations := make([]core.Citation, len(docs))
		for i, d := range docs {
			citations[i] = core.Citation{
				ID:      d.ID,
				Source:  d.Source,
				URI:     d.URI,
				Snippet: d.Text,
			}
		}
		a.Citations = citations
	}

	return passages, nil
}

func (s *Sandwich) runLlm(ctx context.Context, logger core.Logger, q core.Query, passages []string, a *core.Answer) error {
	meter := s.opts.Obs.Meter

	promptData := map[string]interface{}{
		"query":     q.Text,
		"documents": passages,
	}
	if q.Meta != nil {
		for k, v := range q.Meta {
			promptData[k] = v
		}
	}

	tLlmStart := time.Now()
	llmRes, err := s.llm.Complete(ctx, llm.Request{
		Prompt:    q.Text,
		Context:   passages,
		MaxTokens: s.opts.MaxTokens,
		Data:      promptData,
	})
	if err != nil {
		logger.Errorf("llm failed", "error", err)
		return fmt.Errorf("llm failed: %w", err)
	}
	llmMs := time.Since(tLlmStart).Milliseconds()
	a.Text = llmRes.Text
	a.Meta["llm_ms"] = llmMs
	a.Meta["token_usage"] = llmRes.Usage
	if meter != nil {
		meter.Record("manglekit.llm_ms", float64(llmMs))
	}
	return nil
}

func (s *Sandwich) runPostRules(ctx context.Context, logger core.Logger, q core.Query, a *core.Answer) error {
	if s.opts.Rules == nil {
		return nil
	}
	meter := s.opts.Obs.Meter

	tPostRulesStart := time.Now()
	res, err := s.opts.Rules.Evaluate(core.Post, q, a)
	if meter != nil {
		meter.Record("manglekit.rules_post_ms", float64(time.Since(tPostRulesStart).Milliseconds()))
	}
	if err != nil {
		logger.Errorf("post-rules failed", "error", err)
		return fmt.Errorf("post-rules failed: %w", err)
	}
	if !res.Allowed {
		logger.Warnf("request denied by post-rule", "reason", res.Reason)
		return fmt.Errorf("%w: %s", core.ErrDenied, res.Reason)
	}
	if res.Mutate != nil {
		logger.Debugf("answer mutated by post-rule")
		res.Mutate(&q, a)
	}
	return nil
}
