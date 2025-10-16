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
	factory := func(ctx context.Context, deps diapi.EmbedderDeps, cfg any) (ai.Embedder, error) {
		opts, ok := cfg.(*embed.OpenAIEmbedderOptions)
		if !ok {
			return nil, fmt.Errorf("invalid options type, expected *embed.OpenAIEmbedderOptions, got %T", cfg)
		}
		if deps.Genkit == nil {
			return nil, fmt.Errorf("missing required dependency 'genkit'")
		}

		plugin := &oai.OpenAI{
			APIKey: opts.APIKey,
		}
		embedder := plugin.Embedder(deps.Genkit, opts.Model)
		if embedder == nil {
			return nil, fmt.Errorf("failed to get openai embedder %q from genkit", opts.Model)
		}
		return embedder, nil
	}
	r.RegisterEmbedder("openai-embedder", factory)
	r.RegisterEmbedder("groq-embedder", factory)
	r.RegisterOptions("openai-embedder", (*embed.OpenAIEmbedderOptions)(nil))
	r.RegisterOptions("groq-embedder", (*embed.OpenAIEmbedderOptions)(nil))
}

