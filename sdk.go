package manglekit

import (
	"context"
	"fmt"
	"os"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/logger"
	"github.com/duynguyendang/manglekit/retrieve"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"
)

// InitGenkitWithProviders initializes a genkit instance with available LLM provider plugins.
// This function intelligently detects which LLM provider API keys are set in the environment
// and only registers those plugins that are available.
//
// **Important:** At least one LLM provider API key should be set in the environment
// (e.g., GOOGLE_API_KEY) if your configuration uses LLM components.
// If you don't plan to use LLM components, use InitGenkitBasic() instead.
//
// Supported environment variables:
// - GOOGLE_API_KEY: Enables Google GenAI plugin
// - OPENAI_API_KEY: Enables OpenAI plugin (when available)
//
// The function only registers plugins for which API keys are detected, allowing you to:
// - Use different LLM providers in different deployments
// - Avoid initializing unused providers
// - Gracefully handle missing API keys for providers you don't plan to use
//
// Example:
//
//	g := manglekit.InitGenkitWithProviders(ctx)
//	// Only plugins with corresponding API keys will be registered
func InitGenkitWithProviders(ctx context.Context) *genkit.Genkit {
	// Register Google plugin if GOOGLE_API_KEY is set
	if os.Getenv("GOOGLE_API_KEY") != "" {
		return genkit.Init(ctx, genkit.WithPlugins(&googlegenai.GoogleAI{}))
	}

	// If Google key not found, check for other providers
	// For now, return basic genkit. This allows:
	// 1. Config-based loading to work without API keys
	// 2. Other LLM providers (OpenAI, etc.) to be dynamically loaded if needed
	//
	// TODO: Add support for other providers:
	//   if os.Getenv("OPENAI_API_KEY") != "" {
	//       return genkit.Init(ctx, genkit.WithPlugins(&openai.OpenAI{}))
	//   }

	return genkit.Init(ctx)
}

// InitGenkitBasic initializes a basic genkit instance without any provider plugins.
// Use this when you're building an orchestrator that doesn't use LLM components,
// or when you want to avoid requiring LLM API keys to be set.
//
// If your configuration includes LLM components, they will handle plugin registration
// dynamically when needed.
//
// Example:
//
//	g := manglekit.InitGenkitBasic(ctx)
//	// Genkit is ready, but no LLM plugins are pre-registered
func InitGenkitBasic(ctx context.Context) *genkit.Genkit {
	return genkit.Init(ctx)
}

// FromConfig is a high-level function that loads a Manglekit orchestrator from a YAML
// configuration byte slice. It requires a pre-configured registry with all necessary
// component handlers already registered.
//
// This function initializes genkit without pre-loading LLM provider plugins. This allows
// configurations without LLM components to work even if LLM API keys are not set.
// LLM components will dynamically register their plugins as needed.
func FromConfig(ctx context.Context, data []byte, registry *Registry) (core.Orchestrator, retrieve.Updatable, error) {
	l := logger.NewStdLogger()
	obs := core.Observability{Logger: l}

	// Use basic genkit init to avoid requiring LLM API keys when not needed
	g := InitGenkitBasic(ctx)

	// 2. Create a new builder and register all component handlers.
	b, err := NewBuilder(ctx, registry, obs, g)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create new builder: %w", err)
	}

	// The public NewBuilder returns an interface. We need to cast it back to the
	// internal struct type to access the internal fromConfig method.
	internalBuilder, ok := b.(*builder)
	if !ok {
		return nil, nil, fmt.Errorf("internal error: builder is not of the expected type *builder")
	}

	// 3. Build the orchestrator from the configuration.
	orch, up, err := internalBuilder.fromConfig(ctx, data)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build orchestrator from config: %w", err)
	}
	return orch, up, nil
}
