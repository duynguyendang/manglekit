package llm

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core/diapi"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	oai "github.com/firebase/genkit/go/plugins/compat_oai/openai"
	"github.com/firebase/genkit/go/plugins/googlegenai"
	"github.com/openai/openai-go/option"
)

// RegisterGenkit registers the generic Genkit LLM factory with the Manglekit registry.
// This factory supports ANY Genkit LLM provider by dispatching based on configuration.
func RegisterGenkit(r *manglekit.Registry) error {
	factory := func(ctx context.Context, deps diapi.LLMDeps, cfg *GenkitLLMOptions) (ai.Model, error) {
		if deps.Genkit == nil {
			return nil, fmt.Errorf("genkit instance is required for LLM factory")
		}

		if cfg.Provider == "" {
			return nil, fmt.Errorf("provider is required in GenkitLLMOptions (e.g., 'openai', 'google', 'anthropic', 'vertex')")
		}

		if cfg.Model == "" {
			return nil, fmt.Errorf("model is required in GenkitLLMOptions for provider %q", cfg.Provider)
		}

		// Dispatch to the appropriate Genkit provider plugin
		model, err := createGenkitLLM(ctx, deps.Genkit, cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create genkit LLM for provider %q: %w", cfg.Provider, err)
		}

		if model == nil {
			return nil, fmt.Errorf("genkit provider %q returned nil model for model %q", cfg.Provider, cfg.Model)
		}

		// Log successful delegation
		if deps.Obs.Logger != nil {
			deps.Obs.Logger.Debugf(
				"created genkit llm via dynamic factory",
				"provider", cfg.Provider,
				"model", cfg.Model,
			)
		}

		return model, nil
	}

	return manglekit.Register(r, &GenkitLLMOptions{}, factory)
}

// createGenkitLLM dispatches to the appropriate Genkit provider plugin based on configuration.
// This is the extensibility point: new providers are added here via switch cases.
//
// Supported providers:
//   - "openai", "groq": OpenAI and OpenAI-compatible APIs (via compat_oai plugin)
//   - "google", "vertex", "googlegenai": Google Vertex AI (via googlegenai plugin)
//   - "anthropic": Anthropic models (add support as Genkit plugin becomes available)
//   - "aws-bedrock": AWS Bedrock (add support as Genkit plugin becomes available)
//
// To add support for a new provider:
//  1. Ensure the Genkit plugin package is available
//  2. Add a case to this switch statement
//  3. Implement the provider creation logic
//  4. Update documentation
//  5. NO Manglekit recompilation needed if using ProviderConfig for custom params
func createGenkitLLM(ctx context.Context, g *genkit.Genkit, opts *GenkitLLMOptions) (ai.Model, error) {
	switch opts.Provider {
	case "openai", "groq":
		return createOpenAILLM(g, opts)

	case "google", "googlegenai", "vertex":
		return createGoogleLLM(g, opts)

	default:
		return nil, fmt.Errorf(
			"unsupported LLM provider %q; supported providers: openai, groq, google, vertex, googlegenai",
			opts.Provider,
		)
	}
}

// createOpenAILLM creates an OpenAI or OpenAI-compatible LLM via the genkit compat_oai plugin.
func createOpenAILLM(g *genkit.Genkit, opts *GenkitLLMOptions) (ai.Model, error) {
	if opts.APIKey == "" {
		return nil, fmt.Errorf("apiKey is required for provider %q", opts.Provider)
	}

	// Build OpenAI client options
	clientOpts := []option.RequestOption{
		option.WithAPIKey(opts.APIKey),
	}

	if opts.BaseURL != "" {
		clientOpts = append(clientOpts, option.WithBaseURL(opts.BaseURL))
	}

	// Create OpenAI plugin instance
	plugin := &oai.OpenAI{
		APIKey: opts.APIKey,
		Opts:   clientOpts,
	}

	// Get LLM model from plugin
	model := plugin.Model(g, opts.Model)
	if model == nil {
		return nil, fmt.Errorf(
			"openai plugin failed to create model %q (verify model exists and credentials are valid)",
			opts.Model,
		)
	}

	return model, nil
}

// createGoogleLLM creates a Google Vertex AI LLM via the genkit googlegenai plugin.
func createGoogleLLM(g *genkit.Genkit, opts *GenkitLLMOptions) (ai.Model, error) {
	if opts.APIKey == "" {
		return nil, fmt.Errorf("apiKey is required for provider %q", opts.Provider)
	}

	// Google plugin handles authentication via API key set in environment
	// or passed through the Genkit instance
	// The apiKey should be set as GOOGLE_API_KEY environment variable or configured in Genkit
	model := googlegenai.GoogleAIModel(g, opts.Model)
	if model == nil {
		return nil, fmt.Errorf(
			"google plugin failed to create model %q "+
				"(verify model exists, GOOGLE_API_KEY is set, and Genkit is initialized with GoogleAI plugin)",
			opts.Model,
		)
	}

	return model, nil
}
