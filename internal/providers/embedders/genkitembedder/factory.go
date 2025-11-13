package genkitembedder

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core/diapi"
	"github.com/duynguyendang/manglekit/embed"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	oai "github.com/firebase/genkit/go/plugins/compat_oai/openai"
	"github.com/firebase/genkit/go/plugins/googlegenai"
	"github.com/openai/openai-go/option"
)

// Register registers the generic Genkit embedder factory with the Manglekit registry.
// This factory supports ANY Genkit embedder provider by dispatching based on configuration.
func Register(r *manglekit.Registry) error {
	factory := func(ctx context.Context, deps diapi.EmbedderDeps, cfg *embed.GenkitEmbedderOptions) (ai.Embedder, error) {
		if deps.Genkit == nil {
			return nil, fmt.Errorf("genkit instance is required for embedder factory")
		}

		if cfg.Provider == "" {
			return nil, fmt.Errorf("provider is required in GenkitEmbedderOptions (e.g., 'openai', 'google', 'vertex', 'cohere')")
		}

		if cfg.Model == "" {
			return nil, fmt.Errorf("model is required in GenkitEmbedderOptions for provider %q", cfg.Provider)
		}

		// Dispatch to the appropriate Genkit provider plugin
		embedder, err := createGenkitEmbedder(ctx, deps.Genkit, cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create genkit embedder for provider %q: %w", cfg.Provider, err)
		}

		if embedder == nil {
			return nil, fmt.Errorf("genkit provider %q returned nil embedder for model %q", cfg.Provider, cfg.Model)
		}

		// Log successful delegation
		if deps.Obs.Logger != nil {
			deps.Obs.Logger.Debugf(
				"created genkit embedder via dynamic factory",
				"provider", cfg.Provider,
				"model", cfg.Model,
			)
		}

		return embedder, nil
	}

	return manglekit.Register(r, &embed.GenkitEmbedderOptions{}, factory)
}

// createGenkitEmbedder dispatches to the appropriate Genkit provider plugin based on configuration.
// This is the extensibility point: new providers are added here via switch cases.
//
// Supported providers:
//   - "openai", "groq": OpenAI and OpenAI-compatible APIs (via compat_oai plugin)
//   - "google", "vertex", "googlegenai": Google Vertex AI (via googlegenai plugin)
//   - "cohere": Cohere embeddings (add support as Genkit plugin becomes available)
//   - "anthropic": Anthropic models (add support as Genkit plugin becomes available)
//
// To add support for a new provider:
//  1. Ensure the Genkit plugin package is available
//  2. Add a case to this switch statement
//  3. Implement the provider creation logic
//  4. Update documentation
//  5. NO Manglekit recompilation needed if using ProviderConfig for custom params
func createGenkitEmbedder(ctx context.Context, g *genkit.Genkit, opts *embed.GenkitEmbedderOptions) (ai.Embedder, error) {
	switch opts.Provider {
	case "openai", "groq":
		return createOpenAIEmbedder(g, opts)

	case "google", "googlegenai", "vertex":
		return createGoogleEmbedder(g, opts)

	case "cohere":
		return createCohereEmbedder(g, opts)

	default:
		return nil, fmt.Errorf(
			"unsupported embedder provider: %q\n"+
				"Supported providers: openai, google, vertex, cohere, anthropic, etc.\n"+
				"Tip: Ensure the provider's Genkit plugin is initialized in your genkit.Genkit instance",
			opts.Provider,
		)
	}
}

// createOpenAIEmbedder creates an OpenAI or OpenAI-compatible embedder.
// Supports OpenAI and services like Groq that provide OpenAI-compatible APIs.
func createOpenAIEmbedder(g *genkit.Genkit, opts *embed.GenkitEmbedderOptions) (ai.Embedder, error) {
	if opts.APIKey == "" {
		return nil, fmt.Errorf(
			"openai embedder requires apiKey (either in config or OPENAI_API_KEY environment variable)",
		)
	}

	// Build OpenAI client options
	var clientOpts []option.RequestOption
	clientOpts = append(clientOpts, option.WithAPIKey(opts.APIKey))

	if opts.BaseURL != "" {
		clientOpts = append(clientOpts, option.WithBaseURL(opts.BaseURL))
	}

	// Create OpenAI plugin instance
	plugin := &oai.OpenAI{
		APIKey: opts.APIKey,
		Opts:   clientOpts,
	}

	// Get embedder from plugin
	embedder := plugin.Embedder(g, opts.Model)
	if embedder == nil {
		return nil, fmt.Errorf(
			"openai plugin failed to create embedder for model %q (verify model exists and credentials are valid)",
			opts.Model,
		)
	}

	return embedder, nil
}

// createGoogleEmbedder creates a Google Vertex AI embedder via the googlegenai plugin.
// Supports Google's embedding models like "embedding-001" and newer versions.
func createGoogleEmbedder(g *genkit.Genkit, opts *embed.GenkitEmbedderOptions) (ai.Embedder, error) {
	// Google plugin handles authentication via environment variables (GOOGLE_API_KEY)
	// and project configuration
	embedder := googlegenai.GoogleAIEmbedder(g, opts.Model)
	if embedder == nil {
		return nil, fmt.Errorf(
			"google plugin failed to create embedder for model %q "+
				"(verify model exists, GOOGLE_API_KEY is set, and Genkit is initialized with GoogleAI plugin)",
			opts.Model,
		)
	}

	return embedder, nil
}

// createCohereEmbedder creates a Cohere embedder.
// Note: This requires Genkit to expose a Cohere plugin.
// Placeholder implementation for future support.
func createCohereEmbedder(g *genkit.Genkit, opts *embed.GenkitEmbedderOptions) (ai.Embedder, error) {
	// TODO: Implement when Genkit exposes Cohere plugin
	// For now, provide helpful error message
	return nil, fmt.Errorf(
		"cohere embedder support not yet implemented in Manglekit\n" +
			"Status: Waiting for Genkit Cohere plugin availability\n" +
			"Workaround: Use OpenAI or Google embedders",
	)
}
