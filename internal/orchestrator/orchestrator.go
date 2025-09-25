// Package orchestrator implements the main workflow orchestrator for Manglekit.
package orchestrator

import (
	"context"
	"errors"
	"fmt"
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
	MaxContextTokens  int              `yaml:"maxContextTokens"`
	FallbackThreshold float64          `yaml:"fallbackThreshold"`
	LLM               types.LLMConfig  `yaml:"llm"`
	Mangle            mangle.Config    `yaml:"mangle"`
	IntentParser      genintent.Config `yaml:"intentParser"`
}

// orchestrator implements the types.Orchestrator interface.
type orchestrator struct {
	retriever  types.Retriever
	reranker   types.Reranker
	llmGateway types.Gateway
	processor  types.Processor
	intent     types.IntentParser
	config     Config
}

// New creates a new Orchestrator.
func New(ctx context.Context, g *genkit.Genkit, cfg Config, retriever types.Retriever, reranker types.Reranker) (types.Orchestrator, error) {
	if g == nil {
		return nil, errors.New("genkit runtime is required")
	}
	if retriever == nil {
		return nil, errors.New("retriever is required")
	}
	if reranker == nil {
		return nil, errors.New("reranker is required")
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
		config:     cfg,
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

	retrievedChunks, err := o.retriever.Search(ctx, expanded, expanded.Filters)
	if err != nil {
		return nil, fmt.Errorf("hybrid search: %w", err)
	}

	rerankedChunks, rerankExplanations, err := o.reranker.Rerank(ctx, expanded, retrievedChunks)
	if err != nil {
		return nil, fmt.Errorf("rerank results: %w", err)
	}

	postContext, postExplanations := o.processor.PostProcess(rerankedChunks, &types.Context{
		UserContext: input.UserContext,
		Constraints: expanded.Constraints,
	})

	var stageExplanations []types.Explanation
	if rerankExplanations != nil {
		stageExplanations = append(stageExplanations, rerankExplanations...)
	}
	if postExplanations != nil {
		stageExplanations = append(stageExplanations, (*postExplanations)...)
	}

	fallbackTriggered := len(postContext) == 0
	if !fallbackTriggered && o.config.FallbackThreshold > 0 {
		fallbackTriggered = postContext[0].Score < o.config.FallbackThreshold
	}

	if fallbackTriggered {
		response := &types.Response{
			Answer:       "No policy-compliant context met the confidence threshold. Please refine your query or provide more detail.",
			Explanations: append([]types.Explanation(nil), stageExplanations...),
		}
		o.attachIntentMetadata(response, intentResult)
		o.attachMangleExplanation(response, expanded)
		return response, nil
	}

	limitedChunks := o.limitChunks(postContext)
	response, err := o.llmGateway.Generate(ctx, input.Query, limitedChunks)
	if err != nil {
		return nil, err
	}

	response.Explanations = append(response.Explanations, stageExplanations...)

	o.attachIntentMetadata(response, intentResult)
	o.attachMangleExplanation(response, expanded)

	return response, nil
}

func (o *orchestrator) attachIntentMetadata(response *types.Response, intentResult *types.IntentResult) {
	if intentResult == nil || response == nil {
		return
	}
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

func (o *orchestrator) attachMangleExplanation(response *types.Response, expanded *types.ExpandedQuery) {
	if response == nil || expanded == nil {
		return
	}
	if expanded.Explanation != "" {
		response.Explanations = append(response.Explanations, types.Explanation{
			Type:   "mangle-pre",
			Rule:   "expanded_query",
			Action: "applied",
			Reason: expanded.Explanation,
		})
	}
	if response.Metadata == nil {
		response.Metadata = map[string]any{}
	}
	if meta, ok := response.Metadata.(map[string]any); ok {
		meta["expandedQuery"] = expanded
	}
}

func (o *orchestrator) limitChunks(chunks []*types.Chunk) []*types.Chunk {
	maxTokens := o.config.MaxContextTokens
	if maxTokens <= 0 {
		return chunks
	}
	var (
		totalTokens int
		limited     []*types.Chunk
	)
	for _, chunk := range chunks {
		if chunk == nil {
			continue
		}
		tokenCount := len(strings.Fields(chunk.Text))
		if len(limited) > 0 && totalTokens+tokenCount > maxTokens {
			break
		}
		totalTokens += tokenCount
		limited = append(limited, chunk)
		if totalTokens >= maxTokens {
			break
		}
	}
	if len(limited) == 0 {
		return chunks
	}
	return limited
}
