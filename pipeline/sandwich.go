package pipeline

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/duynguyendang/manglekit/core"
	ilogger "github.com/duynguyendang/manglekit/internal/logger"
	"github.com/duynguyendang/manglekit/llm"
	"github.com/duynguyendang/manglekit/rerank"
	"github.com/duynguyendang/manglekit/retrieve"
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
	opts      core.Options
	retriever retrieve.Retriever
	reranker  rerank.Reranker
	llm       llm.Client
	closers   []core.ResourceCloser
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
	s.opts.EnsureLogger(ilogger.NewStdLogger())
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

// Close releases any external resources held by the orchestrator (e.g., API clients).
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

func (s *Sandwich) Run(ctx context.Context, q core.Query) (core.Answer, error) {
	requestID := uuid.New().String()
	logger := s.opts.Obs.Logger.With("request_id", requestID)
	logger.Infof("Pipeline run started", "query", q.Text)

	if s.opts.Obs.Tracer != nil {
		endTrace := s.opts.Obs.Tracer.StartSpan("manglekit.Run")
		defer endTrace()
	}
	answer := core.Answer{
		Meta: map[string]any{},
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
	passages, err := s.prepareLlmRequest(logger, docs, &answer)
	if err != nil {
		return answer, fmt.Errorf("prepare llm request failed: %w", err)
	}

	// 5. LLM
	if err := s.runLlm(ctx, logger, q, passages, &answer); err != nil {
		return answer, err
	}

	// 6. Post-Rules
	if err := s.runPostRules(ctx, logger, q, &answer); err != nil {
		return answer, err
	}

	logger.Infof("Pipeline run finished successfully")
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
		logger.Errorf("Pre-rules failed", "error", err)
		return q, fmt.Errorf("pre-rules failed: %w", err)
	}
	if !res.Allowed {
		if res.Mutate != nil {
			res.Mutate(&q, a)
		}
		logger.Warnf("Request denied by pre-rule", "reason", res.Reason)
		return q, fmt.Errorf("%w: %s", core.ErrDenied, res.Reason)
	}
	if res.Mutate != nil {
		res.Mutate(&q, a)
		logger.Debugf("Query mutated by pre-rule")
	}

	var filters any
	var expansions any
	if q.Meta != nil {
		filters = q.Meta["filters"]
		expansions = q.Meta["expansion_terms"]
	}
	logger.Debugf("Pre-rules outputs", "filters", filters, "expansions", expansions)
	return q, nil
}

func (s *Sandwich) runRetrieve(ctx context.Context, logger core.Logger, q core.Query, a *core.Answer) ([]core.Doc, error) {
	meter := s.opts.Obs.Meter

	tRetrieveStart := time.Now()
	retrReq := retrieve.Request{Query: q.Text, TopK: s.opts.TopK, Meta: q.Meta}
	logger.Infof("Calling retriever", "filters", q.Meta["filters"], "expansions", q.Meta["expansion_terms"])

	retrRes, err := s.retriever.Retrieve(ctx, retrReq)
	if err != nil {
		logger.Errorf("Retrieve failed", "error", err)
		return nil, fmt.Errorf("retrieve failed: %w", err)
	}

	retrieveMs := time.Since(tRetrieveStart).Milliseconds()
	a.Meta["retrieve_ms"] = retrieveMs
	if meter != nil {
		meter.Record("manglekit.retrieve_ms", float64(retrieveMs))
	}

	docs := retrRes.Docs
	a.Meta["original_docs"] = docs
	logger.Infof("Retrieved documents", "count", len(docs))
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
		logger.Errorf("Rerank failed", "error", err)
		return nil, fmt.Errorf("rerank failed: %w", err)
	}
	logger.Infof("Reranked documents", "count", len(rerankedDocs))
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
		logger.Warnf("Fallback threshold not met", "best_score", bestScore, "threshold", s.opts.FallbackThreshold)
		return nil, core.ErrNoEvidence
	}
	return d, nil
}

func (s *Sandwich) prepareLlmRequest(logger core.Logger, docs []core.Doc, a *core.Answer) ([]string, error) {
	logger.Debugf("Preparing LLM request", "doc_count", len(docs))
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

	tLlmStart := time.Now()
	llmRes, err := s.llm.Complete(ctx, llm.Request{
		Prompt:    q.Text,
		Context:   passages,
		MaxTokens: s.opts.MaxTokens,
		Data:      q.Meta,
	})
	if err != nil {
		logger.Errorf("LLM failed", "error", err)
		return fmt.Errorf("llm failed: %w", err)
	}
	llmMs := time.Since(tLlmStart).Milliseconds()
	a.Text = llmRes.Text
	a.Meta["llm_ms"] = llmMs
	a.Meta["token_usage"] = llmRes.Usage
	if meter != nil {
		meter.Record("manglekit.llm_ms", float64(llmMs))
	}
	logger.Infof("LLM completed", "duration_ms", llmMs)
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
		logger.Errorf("Post-rules failed", "error", err)
		return fmt.Errorf("post-rules failed: %w", err)
	}
	if !res.Allowed {
		logger.Warnf("Request denied by post-rule", "reason", res.Reason)
		return fmt.Errorf("%w: %s", core.ErrDenied, res.Reason)
	}
	if res.Mutate != nil {
		logger.Debugf("Answer mutated by post-rule")
		res.Mutate(&q, a)
	}
	return nil
}
