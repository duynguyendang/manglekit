// Package openai provides a MangleKit embedder implementation for OpenAI and
// other services that offer an OpenAI-compatible API (such as Groq).
// Groq can be configured by setting the base_url parameter in the config.yaml.
package openai

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core/diapi"
	"github.com/duynguyendang/manglekit/embed"
	"github.com/firebase/genkit/go/ai"
	oai "github.com/firebase/genkit/go/plugins/compat_oai/openai"
	"github.com/openai/openai-go/option"
)

func Register(r *manglekit.Registry) error {
	// Register OpenAI Embedder
	if err := manglekit.Register(r, &embed.OpenAIEmbedderOptions{},
		func(ctx context.Context, deps diapi.EmbedderDeps, cfg *embed.OpenAIEmbedderOptions) (ai.Embedder, error) {
			if deps.Genkit == nil {
				return nil, fmt.Errorf("missing required dependency 'genkit'")
			}
			if cfg.APIKey == "" {
				return nil, fmt.Errorf("apiKey is required for openai embedder")
			}
			var opts []option.RequestOption
			if cfg.BaseURL != "" {
				opts = append(opts, option.WithBaseURL(cfg.BaseURL))
			}
			plugin := &oai.OpenAI{APIKey: cfg.APIKey, Opts: opts}
			var embedder ai.Embedder
			if !cfg.SkipModelCheck {
				embedder = plugin.Embedder(deps.Genkit, cfg.Model)
				if embedder == nil {
					return nil, fmt.Errorf("failed to get openai embedder %q from genkit", cfg.Model)
				}
			}
			return embedder, nil
		},
	); err != nil {
		return fmt.Errorf("failed to register openai embedder: %w", err)
	}

	return nil
}
