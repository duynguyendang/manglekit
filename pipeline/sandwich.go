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
// It satisfies a requirement of the `core.Orchestrator` interface, allowing access
// to the retriever instance at runtime. This is particularly useful for calling
// methods on an `Updatable` retriever to add or modify documents in a live system.
// The return type is `any` to avoid circular package dependencies; the caller is
// expected to perform a type assertion to `retrieve.Retriever` or `retrieve.Updatable`.
func (s *Sandwich) Retriever() any {
	return s.retriever
}

// Run executes the full, sequential orchestration pipeline for a given query.
// It is the primary method of the `Sandwich` orchestrator and follows a series
// of distinct steps, with observability hooks for logging and tracing at each point.
//
// The process is as follows:
//  1. **Pre-Rules**: Executes Mangle rules at the 'pre' stage to validate,
//     normalize, or modify the incoming query. This can result in adding filters
//     or expansion terms to the query's metadata. If a rule denies the request,
//     the pipeline halts immediately.
//  2. **Retrieve**: Fetches an initial set of documents using the configured
//     retriever, passing along any metadata from the pre-rules stage.
//  3. **Rerank**: If a reranker is configured, it re-scores and re-orders the
//     retrieved documents to improve their relevance to the query.
//  4. **Fallback Check**: If a `FallbackThreshold` is set, the pipeline checks the
//     top score from the reranker. If the score is below the threshold, it
//     exits early with an `ErrNoEvidence` error, preventing the LLM from being
//     called with low-quality context.
//  5. **LLM Call**: Sends the final, reranked context and the user prompt to the
//     language model to generate a synthesized answer.
//  6. **Post-Rules**: Executes Mangle rules at the 'post' stage. These rules can
//     inspect the generated answer and the source citations, filtering or modifying
//     them to enforce policies (e.g., removing PII, checking for entitlements).
//     If a rule denies the result, the pipeline halts.
//
// ctx is the context for the entire operation, used for cancellation and tracing.
// q is the user's query to be processed.
// It returns the final `core.Answer` or an error if any stage of the pipeline fails.
func (s *Sandwich) Run(ctx context.Context, q core.Query) (core.Answer, error) {
	// 0. Setup observability
	logger := s.opts.Obs.Logger
	meter := s.opts.Obs.Meter
	if logger != nil {
		logger.Info("pipeline run started", "query", q.Text)
	} else {
		fmt.Printf("[sandwich] pipeline run started query=%q\n", q.Text)
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
			} else {
				fmt.Printf("[sandwich] pre-rules failed: %v\n", err)
			}
			return core.Answer{}, fmt.Errorf("pre-rules failed: %w", err)
		}
		if !res.Allowed {
			if res.Mutate != nil {
				res.Mutate(&q, &answer)
			}
			if logger != nil {
				logger.Info("request denied by pre-rule", "reason", res.Reason)
			} else {
				fmt.Printf("[sandwich] request denied by pre-rule reason=%q\n", res.Reason)
			}
			return answer, fmt.Errorf("%w: %s", core.ErrDenied, res.Reason)
		}
		if res.Mutate != nil {
			// ÁP DỤNG MUTATE để filters/expansion_terms đi vào q.Meta
			res.Mutate(&q, &answer)
			if logger != nil {
				logger.Info("query mutated by pre-rule")
			} else {
				fmt.Printf("[sandwich] query mutated by pre-rule\n")
			}
		}

		// Log filters/expansions sau mutate (dù có logger hay không)
		var filters any
		var expansions any
		if q.Meta != nil {
			filters = q.Meta["filters"]
			expansions = q.Meta["expansion_terms"]
		}
		if logger != nil {
			logger.Info("pre-rules outputs", "filters", filters, "expansions", expansions)
		} else {
			fmt.Printf("[sandwich] pre-rules outputs filters=%#v expansions=%#v\n", filters, expansions)
		}
	}

	// 2. Retrieve
	tRetrieveStart := time.Now()
	retrReq := retrieve.Request{Query: q.Text, TopK: s.opts.TopK, Meta: q.Meta}
	if logger == nil { // in ra trước khi gọi retriever
		fmt.Printf("[sandwich] calling retriever with filters=%#v expansions=%#v\n",
			q.Meta["filters"], q.Meta["expansion_terms"])
	}
	retrRes, err := s.retriever.Retrieve(retrReq)
	if err != nil {
		if logger != nil {
			logger.Error("retrieve failed", "error", err)
		} else {
			fmt.Printf("[sandwich] retrieve failed: %v\n", err)
		}
		return core.Answer{}, fmt.Errorf("retrieve failed: %w", err)
	}
	retrieveMs := time.Since(tRetrieveStart).Milliseconds()
	answer.Meta["retrieve_ms"] = retrieveMs
	if meter != nil {
		meter.Record("manglekit.retrieve_ms", float64(retrieveMs))
	}
	docs := retrRes.Docs

	// Lưu original_docs NGAY sau retrieve để post-rules có thể inspect
	answer.Meta["original_docs"] = docs

	if logger != nil {
		logger.Info("retrieved documents", "count", len(docs))
	} else {
		fmt.Printf("[sandwich] retrieved %d docs\n", len(docs))
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
			} else {
				fmt.Printf("[sandwich] rerank failed: %v\n", err)
			}
			return core.Answer{}, fmt.Errorf("rerank failed: %w", err)
		}
		if logger != nil {
			logger.Info("reranked documents", "count", len(rerankedDocs))
		} else {
			fmt.Printf("[sandwich] reranked documents count=%d\n", len(rerankedDocs))
		}
		d := make([]core.Doc, len(rerankedDocs))
		citations := make([]core.Citation, len(rerankedDocs))
		if len(rerankedDocs) > 0 {
			bestScore = rerankedDocs[0].Score
		}
		for i, rd := range rerankedDocs {
			d[i] = rd.Doc
			citations[i] = core.Citation{
				ID:      rd.Doc.ID,
				Source:  rd.Doc.Source,
				URI:     rd.Doc.URI,
				Snippet: rd.Doc.Text,
				Score:   rd.Score,
			}
		}
		docs = d
		answer.Citations = citations
	}
	answer.Meta["best_score"] = bestScore

	// 4. Fallback Threshold
	if s.opts.FallbackThreshold > 0 && bestScore < s.opts.FallbackThreshold {
		if logger != nil {
			logger.Info("fallback threshold not met", "best_score", bestScore, "threshold", s.opts.FallbackThreshold)
		} else {
			fmt.Printf("[sandwich] fallback threshold not met best_score=%.4f threshold=%.4f\n", bestScore, s.opts.FallbackThreshold)
		}
		return core.Answer{}, core.ErrNoEvidence
	}

	// 5. LLM
	passages := make([]string, len(docs))
	for i, d := range docs {
		passages[i] = d.Text
	}
	tLlmStart := time.Now()
	llmRes, err := s.llm.Complete(llm.Request{
		Prompt:    q.Text,
		Context:   passages,
		MaxTokens: s.opts.MaxTokens,
		Data:      q.Meta,
	})
	if err != nil {
		if logger != nil {
			logger.Error("llm failed", "error", err)
		} else {
			fmt.Printf("[sandwich] llm failed: %v\n", err)
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
		// KHÔNG ghi đè original_docs ở đây nữa (đã lưu sau retrieve)
		tPostRulesStart := time.Now()
		res, err := s.opts.Rules.Evaluate(core.Post, q, &answer)
		if meter != nil {
			meter.Record("manglekit.rules_post_ms", float64(time.Since(tPostRulesStart).Milliseconds()))
		}
		if err != nil {
			if logger != nil {
				logger.Error("post-rules failed", "error", err)
			} else {
				fmt.Printf("[sandwich] post-rules failed: %v\n", err)
			}
			return core.Answer{}, fmt.Errorf("post-rules failed: %w", err)
		}
		if !res.Allowed {
			if logger != nil {
				logger.Info("request denied by post-rule", "reason", res.Reason)
			} else {
				fmt.Printf("[sandwich] request denied by post-rule reason=%q\n", res.Reason)
			}
			return core.Answer{}, fmt.Errorf("%w: %s", core.ErrDenied, res.Reason)
		}
		if res.Mutate != nil {
			if logger != nil {
				logger.Info("answer mutated by post-rule")
			} else {
				fmt.Printf("[sandwich] answer mutated by post-rule\n")
			}
			res.Mutate(&q, &answer)
		}
	}

	if logger != nil {
		logger.Info("pipeline run finished successfully")
	} else {
		fmt.Printf("[sandwich] pipeline run finished successfully\n")
	}
	return answer, nil
}
