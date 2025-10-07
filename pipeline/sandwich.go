package pipeline

import (
	"context"
	"fmt"
	"time"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/llm"
	"github.com/duynguyendang/manglekit/rerank"
	"github.com/duynguyendang/manglekit/retrieve"
)

// Sandwich implements the default MangleKit orchestrator. It follows a
// "sandwich" pattern: Pre-retrieval rules, Retrieval, Reranking, an LLM call,
// and finally Post-retrieval rules.
type Sandwich struct {
	opts      core.Options
	retriever retrieve.Retriever
	reranker  rerank.Reranker
	llm       llm.Client
}

// NewSandwich creates a new Sandwich orchestrator. It takes core.Options,
// performs type assertions to ensure the provided components (Retriever, LLM,
// Reranker) are of the correct type, and initializes the orchestrator.
//
// o contains the configuration and components for the pipeline.
// It returns a configured core.Orchestrator or an error if any component
// has an invalid type.
func NewSandwich(o core.Options) (core.Orchestrator, error) {
	s := &Sandwich{opts: o}
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
// It satisfies a requirement of the core.Orchestrator interface, allowing access
// to the retriever instance at runtime, for example, to use its Update methods.
func (s *Sandwich) Retriever() any {
	return s.retriever
}

// Run executes the full orchestration pipeline for a given query.
// The process is as follows:
// 1. Pre-Rules: Executes Mangle rules at the 'pre' stage to validate,
//    normalize, or modify the incoming query.
// 2. Retrieve: Fetches relevant documents using the configured retriever.
// 3. Rerank: If a reranker is configured, it re-scores and re-orders the
//    retrieved documents for better relevance.
// 4. Fallback Check: If a confidence threshold is set and the best score from
//    reranking is too low, it can exit early.
// 5. LLM: Sends the final context and prompt to the language model to generate an answer.
// 6. Post-Rules: Executes Mangle rules at the 'post' stage to filter or
//    modify the final answer and its citations based on policies.
//
// ctx is the context for the entire operation.
// q is the user's query to be processed.
// It returns the final Answer or an error if any stage of the pipeline fails.
func (s *Sandwich) Run(ctx context.Context, q core.Query) (core.Answer, error) {
	// 0. Setup observability
	logger := s.opts.Obs.Logger
	meter := s.opts.Obs.Meter
	if logger != nil {
		logger.Info("pipeline run started", "query", q.Text)
	}

	var endTrace func(attrs ...any)
	if s.opts.Obs.Tracer != nil {
		endTrace = s.opts.Obs.Tracer.StartSpan("manglekit.Run")
		defer endTrace()
	}
	answer := core.Answer{
		Meta: map[string]any{},
	}

	// 1. Pre-Rules
	if s.opts.Rules != nil {
		tPreRulesStart := time.Now()
		res, err := s.opts.Rules.Evaluate(core.Pre, q, nil)
		if meter != nil {
			meter.Record("manglekit.rules_pre_ms", float64(time.Since(tPreRulesStart).Milliseconds()))
		}
		if err != nil {
			if logger != nil {
				logger.Error("pre-rules failed", "error", err)
			}
			return core.Answer{}, fmt.Errorf("pre-rules failed: %w", err)
		}
		if !res.Allowed {
			if res.Mutate != nil {
				res.Mutate(&q, &answer)
			}
			if logger != nil {
				logger.Info("request denied by pre-rule", "reason", res.Reason)
			}
			return answer, fmt.Errorf("%w: %s", core.ErrDenied, res.Reason)
		}
		if res.Mutate != nil {
			if logger != nil {
				logger.Info("query mutated by pre-rule")
			}
			res.Mutate(&q, &answer)
		}
	}

	// 2. Retrieve
	tRetrieveStart := time.Now()
	retrReq := retrieve.Request{Query: q.Text, TopK: s.opts.TopK, Meta: q.Meta}
	retrRes, err := s.retriever.Retrieve(retrReq)
	if err != nil {
		if logger != nil {
			logger.Error("retrieve failed", "error", err)
		}
		return core.Answer{}, fmt.Errorf("retrieve failed: %w", err)
	}
	retrieveMs := time.Since(tRetrieveStart).Milliseconds()
	answer.Meta["retrieve_ms"] = retrieveMs
	if meter != nil {
		meter.Record("manglekit.retrieve_ms", float64(retrieveMs))
	}
	docs := retrRes.Docs
	if logger != nil {
		logger.Info("retrieved documents", "count", len(docs))
	}

	// 3. Rerank
	bestScore := 0.0
	if s.reranker != nil {
		tRerankStart := time.Now()
		rerankedDocs, err := s.reranker.Rerank(rerank.Request{Query: q.Text, Docs: docs, TopK: s.opts.TopK})
		if meter != nil {
			meter.Record("manglekit.rerank_ms", float64(time.Since(tRerankStart).Milliseconds()))
		}
		if err != nil {
			if logger != nil {
				logger.Error("rerank failed", "error", err)
			}
			return core.Answer{}, fmt.Errorf("rerank failed: %w", err)
		}
		if logger != nil {
			logger.Info("reranked documents", "count", len(rerankedDocs))
		}
		docs = make([]core.Doc, len(rerankedDocs))
		citations := make([]core.Citation, len(rerankedDocs))
		if len(rerankedDocs) > 0 {
			bestScore = rerankedDocs[0].Score
		}
		for i, rd := range rerankedDocs {
			docs[i] = rd.Doc
			citations[i] = core.Citation{
				ID:      rd.Doc.ID,
				Source:  rd.Doc.Source,
				URI:     rd.Doc.URI,
				Snippet: rd.Doc.Text,
				Score:   rd.Score,
			}
		}
		answer.Citations = citations
	}
	answer.Meta["best_score"] = bestScore

	// 4. Fallback Threshold
	if s.opts.FallbackThreshold > 0 && bestScore < s.opts.FallbackThreshold {
		if logger != nil {
			logger.Info("fallback threshold not met", "best_score", bestScore, "threshold", s.opts.FallbackThreshold)
		}
		return core.Answer{}, core.ErrNoEvidence
	}

	// 5. LLM
	passages := make([]string, len(docs))
	for i, d := range docs {
		passages[i] = d.Text
	}
	tLlmStart := time.Now()
	llmRes, err := s.llm.Complete(llm.Request{Prompt: q.Text, Context: passages, MaxTokens: s.opts.MaxTokens, Data: q.Meta})
	if err != nil {
		if logger != nil {
			logger.Error("llm failed", "error", err)
		}
		return core.Answer{}, fmt.Errorf("llm failed: %w", err)
	}
	llmMs := time.Since(tLlmStart).Milliseconds()
	answer.Text = llmRes.Text
	answer.Meta["llm_ms"] = llmMs
	answer.Meta["token_usage"] = llmRes.Usage
	if meter != nil {
		meter.Record("manglekit.llm_ms", float64(llmMs))
	}

	// 6. Post-Rules
	if s.opts.Rules != nil {
		if answer.Meta == nil {
			answer.Meta = make(map[string]any)
		}
		answer.Meta["original_docs"] = docs
		tPostRulesStart := time.Now()
		res, err := s.opts.Rules.Evaluate(core.Post, q, &answer)
		if meter != nil {
			meter.Record("manglekit.rules_post_ms", float64(time.Since(tPostRulesStart).Milliseconds()))
		}
		if err != nil {
			if logger != nil {
				logger.Error("post-rules failed", "error", err)
			}
			return core.Answer{}, fmt.Errorf("post-rules failed: %w", err)
		}
		if !res.Allowed {
			if logger != nil {
				logger.Info("request denied by post-rule", "reason", res.Reason)
			}
			return core.Answer{}, fmt.Errorf("%w: %s", core.ErrDenied, res.Reason)
		}
		if res.Mutate != nil {
			if logger != nil {
				logger.Info("answer mutated by post-rule")
			}
			res.Mutate(&q, &answer)
		}
	}

	if logger != nil {
		logger.Info("pipeline run finished successfully")
	}
	return answer, nil
}