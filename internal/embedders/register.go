package embedders

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core/diapi"
	"github.com/duynguyendang/manglekit/embed"
	"github.com/duynguyendang/manglekit/internal/embedders/google"
	"github.com/firebase/genkit/go/ai"
	oai "github.com/firebase/genkit/go/plugins/compat_oai/openai"
)

// Register registers all embedder providers and the embedder kind handler.
func Register(r *manglekit.Registry) {
	// Google
	manglekit.Register(r, embed.GoogleEmbedderOptions{},
		func(ctx context.Context, deps diapi.EmbedderDeps, cfg embed.GoogleEmbedderOptions) (ai.Embedder, error) {
			return google.New(cfg, deps.Genkit)
		},
	)
	// OpenAI
	manglekit.Register(r, embed.OpenAIEmbedderOptions{},
		func(ctx context.Context, deps diapi.EmbedderDeps, cfg embed.OpenAIEmbedderOptions) (ai.Embedder, error) {
			plugin := &oai.OpenAI{APIKey: cfg.APIKey}
			embedder := plugin.Embedder(deps.Genkit, cfg.Model)
			if embedder == nil {
				return nil, fmt.Errorf("failed to get openai embedder %q from genkit", cfg.Model)
			}
			return embedder, nil
		},
	)
	// Groq
	manglekit.Register(r, embed.GroqEmbedderOptions{},
		func(ctx context.Context, deps diapi.EmbedderDeps, cfg embed.GroqEmbedderOptions) (ai.Embedder, error) {
			plugin := &oai.OpenAI{APIKey: cfg.APIKey}
			embedder := plugin.Embedder(deps.Genkit, cfg.Model)
			if embedder == nil {
				return nil, fmt.Errorf("failed to get groq embedder %q from genkit", cfg.Model)
			}
			return embedder, nil
		},
	)

	r.RegisterHandler(&Handler{})
}
