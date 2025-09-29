// Package orchestrator implements the main workflow orchestrator for Manglekit.
package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/firebase/genkit/go/genkit"
	"ndduy.dev/manglekit/internal/genintent"
	"ndduy.dev/manglekit/internal/llm"
	"ndduy.dev/manglekit/internal/mangle"
	"ndduy.dev/manglekit/internal/types"
)

// Config holds the configuration for the Orchestrator.
type Config struct {
	MaxContextTokens  int                   `yaml:"maxContextTokens"`
	FallbackThreshold float64               `yaml:"fallbackThreshold"`
	LLM               types.LLMConfig       `yaml:"llm"`
	Mangle            mangle.Config         `yaml:"mangle"`
	IntentParser      genintent.Config      `yaml:"intentParser"`
	Retrieval         types.RetrievalConfig `yaml:"retrieval"`
	Reranker          types.RerankConfig    `yaml:"reranker"`
}

// orchestrator implements the types.Orchestrator interface.
type orchestrator struct {
	retriever  hybridRetriever
	reranker   docReranker
	llmGateway types.Gateway
	processor  types.Processor
	intent     types.IntentParser
	cfg        Config
}

type hybridRetriever interface {
	Retrieve(ctx context.Context, query string, bm25Cfg types.BM25Config, denseCfg types.DenseConfig) ([]string, error)
}

type docReranker interface {
	Rerank(ctx context.Context, query string, docs []string, cfg types.RerankConfig) ([]string, error)
}

// New creates a new Orchestrator.
func New(
	ctx context.Context,
	g *genkit.Genkit,
	cfg Config,
	retriever hybridRetriever,
	reranker docReranker,
) (types.Orchestrator, error) {
	if g == nil {
		return nil, errors.New("genkit runtime is required")
	}

	llmGateway, err := llm.New(ctx, cfg.LLM)
	if err != nil {
		return nil, err
	}

	processor, err := mangle.New(ctx, cfg.Mangle)
	if err != nil {
		return nil, fmt.Errorf("create mangle processor: %w", err)
	}

	intentParser, err := genintent.New(g, cfg.IntentParser)
	if err != nil {
		return nil, fmt.Errorf("create intent parser: %w", err)
	}

	return &orchestrator{
		retriever:  retriever,
		reranker:   reranker,
		llmGateway: llmGateway,
		processor:  processor,
		intent:     intentParser,
		cfg:        cfg,
	}, nil
}

// RunFlow executes the complete Sandwich pattern workflow.
func (o *orchestrator) RunFlow(ctx context.Context, input *types.QueryInput) (*types.Response, error) {
	intentResult, err := o.intent.Parse(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("parse intent: %w", err)
	}
	input.Intent = intentResult

	expanded, err := o.processor.PreProcess(input)
	if err != nil {
		return nil, err
	}

	ragQueryParts := []string{expanded.NormalizedQuery}
	if len(expanded.ExpansionTerms) > 0 {
		ragQueryParts = append(ragQueryParts, expanded.ExpansionTerms...)
	}
	if intentResult != nil {
		keys := make([]string, 0, len(intentResult.Entities))
		for key := range intentResult.Entities {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			ragQueryParts = append(ragQueryParts, intentResult.Entities[key]...)
		}
	}
	ragQuery := strings.Join(ragQueryParts, " ")

	// Hybrid Retrieval
	results, err := o.retriever.Retrieve(ctx, ragQuery, o.cfg.Retrieval.Hybrid.BM25, o.cfg.Retrieval.Hybrid.Dense)
	if err != nil {
		return nil, fmt.Errorf("hybrid retrieval failed: %w", err)
	}

	// Reranking
	rerankedDocs, err := o.reranker.Rerank(ctx, ragQuery, results, o.cfg.Reranker)
	if err != nil {
		return nil, fmt.Errorf("reranking failed: %w", err)
	}

	var chunks []*types.Chunk
	for _, r := range rerankedDocs {
		chunks = append(chunks, &types.Chunk{
			Text: r,
		})
	}

	postChunks, postExplanations := o.processor.PostProcess(chunks, &types.Context{UserContext: input.UserContext})

	var response *types.Response
	if len(postChunks) == 0 {
		response = &types.Response{
			Answer: "I could not find a relevant answer based on the information I have. Please try rephrasing your question.",
			Metadata: map[string]any{
				"fallback": "no_context_after_postprocessing",
			},
			Explanations: []types.Explanation{},
		}
	} else {
		llmResponse, err := o.llmGateway.Generate(ctx, input.Query, postChunks)
		if err != nil {
			return nil, err
		}
		response = llmResponse
	}

	if postExplanations != nil {
		response.Explanations = append(response.Explanations, *postExplanations...)
	}

	if intentResult != nil {
		exp := types.Explanation{
			Type:      "intent",
			Rule:      "genkit-intent",
			Action:    intentResult.Intent,
			Reason:    intentResult.Explanation,
			Timestamp: time.Now(),
		}
		response.Explanations = append(response.Explanations, exp)
		if response.Metadata == nil {
			response.Metadata = map[string]any{}
		}
		if meta, ok := response.Metadata.(map[string]any); ok {
			meta["intent"] = intentResult
		}
	}

	if expanded != nil && expanded.Explanation != "" {
		response.Explanations = append(response.Explanations, types.Explanation{
			Type:   "mangle-pre",
			Rule:   "expanded_query",
			Action: "applied",
			Reason: expanded.Explanation,
		})
	}

	return response, nil
}
