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
	MaxContextTokens  int              `yaml:"maxContextTokens"`
	FallbackThreshold float64          `yaml:"fallbackThreshold"`
	LLM               types.LLMConfig  `yaml:"llm"`
	Mangle            mangle.Config    `yaml:"mangle"`
	IntentParser      genintent.Config `yaml:"intentParser"`
}

// orchestrator implements the types.Orchestrator interface.
type orchestrator struct {
	retriever  types.Retriever
	llmGateway types.Gateway
	rag        ragRetriever
	processor  types.Processor
	intent     types.IntentParser
}

type ragRetriever interface {
	Retrieve(ctx context.Context, query string) ([]string, error)
}

// New creates a new Orchestrator.
func New(ctx context.Context, g *genkit.Genkit, cfg Config, retriever types.Retriever, rag ragRetriever) (types.Orchestrator, error) {
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
		llmGateway: llmGateway,
		rag:        rag,
		processor:  processor,
		intent:     intentParser,
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

	results, err := o.rag.Retrieve(ctx, ragQuery)
	if err != nil {
		return nil, err
	}

	var chunks []*types.Chunk
	for _, r := range results {
		chunks = append(chunks, &types.Chunk{
			Text: r,
		})
	}

	postChunks, _ := o.processor.PostProcess(chunks, &types.Context{UserContext: input.UserContext})
	response, err := o.llmGateway.Generate(ctx, input.Query, postChunks)
	if err != nil {
		return nil, err
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
