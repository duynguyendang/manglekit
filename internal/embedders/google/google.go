// Package google provides a MangleKit embedder implementation that uses Google's
// generative AI models (e.g., embedding-001) via the `google/generative-ai-go` SDK.
package google

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core/diapi"
	"github.com/duynguyendang/manglekit/embed"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"
)

const (
	defaultEmbeddingModel = "embedding-001"
	defaultDim            = 768
)

func Register(r *manglekit.Registry) error {
	if err := manglekit.Register(r, &embed.GoogleEmbedderOptions{},
		func(ctx context.Context, deps diapi.EmbedderDeps, cfg *embed.GoogleEmbedderOptions) (ai.Embedder, error) {
			if deps.Genkit == nil {
				return nil, fmt.Errorf("missing required dependency 'genkit'")
			}
			return New(*cfg, deps.Genkit)
		},
	); err != nil {
		return fmt.Errorf("failed to register google embedder: %w", err)
	}
	return nil
}

// New is the constructor for the GoogleEmbedder. It is registered with the
// MangleKit framework and called by the builder to create a new instance.
//
// opts provides the configuration, such as the specific model to use.
// g is the pre-configured `genkit.Genkit` instance, injected by the builder.
// It returns an `ai.Embedder` or an error if the configuration is invalid.
func New(opts embed.GoogleEmbedderOptions, g *genkit.Genkit) (ai.Embedder, error) {
	if g == nil {
		return nil, fmt.Errorf("google: genkit.Genkit is required")
	}

	modelName := opts.Model
	if modelName == "" {
		modelName = defaultEmbeddingModel
	}

	embedder := googlegenai.GoogleAIEmbedder(g, modelName)
	if embedder == nil {
		return nil, fmt.Errorf("google: failed to create embedder '%s'", modelName)
	}

	return embedder, nil
}
