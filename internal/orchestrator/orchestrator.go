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
	"go.uber.org/zap"
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
	log        *zap.Logger
}

type hybridRetriever interface {
	Retrieve(ctx context.Context, query string, filters map[string]string, bm25Cfg types.BM25Config, denseCfg types.DenseConfig) ([]string, error)
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
	log *zap.Logger,
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
		log:        log,
	}, nil
}

// RunFlow executes the complete Sandwich pattern workflow.
func (o *orchestrator) RunFlow(ctx context.Context, input *types.QueryInput) (*types.Response, error) {
	o.log.Info("starting flow", zap.String("query", input.Query))
	intentResult, err := o.intent.Parse(ctx, input)
	if err != nil {
		o.log.Error("failed to parse intent", zap.Error(err))
		return nil, fmt.Errorf("parse intent: %w", err)
	}
	input.Intent = intentResult
	o.log.Info("parsed intent", zap.Any("intent", intentResult))

	expanded, err := o.processor.PreProcess(input)
	if err != nil {
		o.log.Error("failed to preprocess query", zap.Error(err))
		return nil, err
	}
	o.log.Info("expanded query", zap.Any("expanded", expanded))

	ragQuery := o.constructRAGQuery(expanded, intentResult)
	o.log.Info("constructed RAG query", zap.String("ragQuery", ragQuery))

	// Hybrid Retrieval
	results, err := o.retriever.Retrieve(ctx, ragQuery, expanded.Filters, o.cfg.Retrieval.Hybrid.BM25, o.cfg.Retrieval.Hybrid.Dense)
	if err != nil {
		o.log.Error("hybrid retrieval failed", zap.Error(err))
		return nil, fmt.Errorf("hybrid retrieval failed: %w", err)
	}
	o.log.Info("retrieved documents", zap.Int("count", len(results)))

	// Reranking
	rerankedDocs, err := o.reranker.Rerank(ctx, ragQuery, results, o.cfg.Reranker)
	if err != nil {
		o.log.Error("reranking failed", zap.Error(err))
		return nil, fmt.Errorf("reranking failed: %w", err)
	}
	o.log.Info("reranked documents", zap.Int("count", len(rerankedDocs)))

	var chunks []*types.Chunk
	for _, r := range rerankedDocs {
		chunks = append(chunks, &types.Chunk{
			Text: r,
		})
	}

	postChunks, postExplanations := o.processor.PostProcess(chunks, &types.Context{UserContext: input.UserContext})
	o.log.Info("post-processed chunks", zap.Int("retained_count", len(postChunks)))

	var response *types.Response
	if len(postChunks) == 0 {
		o.log.Warn("no chunks left after post-processing, returning fallback answer")
		response = &types.Response{
			Answer: "I could not find a relevant answer based on the information I have. Please try rephrasing your question.",
			Metadata: map[string]any{
				"fallback": "no_context_after_postprocessing",
			},
			Explanations: []types.Explanation{},
		}
	} else {
		o.log.Info("generating final answer", zap.Int("chunks", len(postChunks)))
		llmResponse, err := o.llmGateway.Generate(ctx, input.Query, postChunks)
		if err != nil {
			o.log.Error("failed to generate response from LLM", zap.Error(err))
			return nil, err
		}
		response = llmResponse
	}

	o.addExplanations(response, postExplanations, intentResult, expanded)
	o.log.Info("flow completed successfully")
	return response, nil
}

func (o *orchestrator) constructRAGQuery(expanded *types.ExpandedQuery, intentResult *types.IntentResult) string {
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
	return strings.Join(ragQueryParts, " ")
}

func (o *orchestrator) addExplanations(
	response *types.Response,
	postExplanations *[]types.Explanation,
	intentResult *types.IntentResult,
	expanded *types.ExpandedQuery,
) {
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
}