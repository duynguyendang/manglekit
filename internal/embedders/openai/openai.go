// Package openai provides a MangleKit embedder implementation for OpenAI and
// other services that offer an OpenAI-compatible API (like Groq).
package openai

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core/diapi"
	"github.com/duynguyendang/manglekit/embed"
	"github.com/firebase/genkit/go/ai"
	oai "github.com/firebase/genkit/go/plugins/compat_oai/openai"
)

func Register(r *manglekit.Registry) {
	must := func(err error) {
		if err != nil {
			panic(err)
		}
	}

	// Register OpenAI Embedder
	must(manglekit.Register(r, &embed.OpenAIEmbedderOptions{},
		func(ctx context.Context, deps diapi.EmbedderDeps, cfg *embed.OpenAIEmbedderOptions) (ai.Embedder, error) {
			if deps.Genkit == nil {
				return nil, fmt.Errorf("missing required dependency 'genkit'")
			}
			plugin := &oai.OpenAI{APIKey: cfg.APIKey}
			embedder := plugin.Embedder(deps.Genkit, cfg.Model)
			if embedder == nil {
				return nil, fmt.Errorf("failed to get openai embedder %q from genkit", cfg.Model)
			}
			return embedder, nil
		},
	))

	// Register Groq Embedder
	must(manglekit.Register(r, &embed.GroqEmbedderOptions{},
		func(ctx context.Context, deps diapi.EmbedderDeps, cfg *embed.GroqEmbedderOptions) (ai.Embedder, error) {
			if deps.Genkit == nil {
				return nil, fmt.Errorf("missing required dependency 'genkit'")
			}
			plugin := &oai.OpenAI{APIKey: cfg.APIKey}
			embedder := plugin.Embedder(deps.Genkit, cfg.Model)
			if embedder == nil {
				return nil, fmt.Errorf("failed to get groq embedder %q from genkit", cfg.Model)
			}
			return embedder, nil
		},
	))
}
