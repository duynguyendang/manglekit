package sdk

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/logger"
	"github.com/duynguyendang/manglekit/providers/all"
)

// Load is a high-level function that loads a Manglekit orchestrator from a YAML
// configuration byte slice. It handles registry creation and component handler
// registration automatically.
//
// This function demonstrates the simplest API for loading a Manglekit orchestrator
// from configuration. All LLM provider plugins are automatically initialized.
func Load(ctx context.Context, data []byte) (core.Orchestrator, error) {
	registry := manglekit.NewRegistry()
	all.Register(registry)
	orch, _, err := manglekit.FromConfig(ctx, data, registry)
	return orch, err
}

// LoadWithRegistry is a high-level function that loads a Manglekit orchestrator from a YAML
// configuration byte slice. It uses a pre-configured registry.
func LoadWithRegistry(ctx context.Context, data []byte, registry *manglekit.Registry) (core.Orchestrator, error) {
	orch, _, err := manglekit.FromConfig(ctx, data, registry)
	return orch, err
}

// NewBuilder provides a programmatic entry point for constructing a Manglekit orchestrator.
// It initializes a new builder with a default registry and observability, allowing for
// fluent, code-based pipeline configuration.
//
// This function automatically:
// - Creates a new registry and registers all standard providers
// - Initializes genkit with all LLM provider plugins via InitGenkitWithProviders
// - Creates observability infrastructure (logging)
// - Returns a builder ready for fluent configuration
//
// **Note:** This function requires LLM API keys if you're using LLM components.
// For config-based loading without pre-initializing LLM plugins, use Load() instead.
//
// Example usage:
//
//	builder, err := sdk.NewBuilder(ctx)
//	if err != nil {
//		log.Fatal(err)
//	}
//	builder.WithOptions("bm25", &bm25.BM25Options{Path: "docs"}).
//		WithOptions("google", &llm.GoogleOptions{Model: "gemini-1.5-flash"})
//	orch, _, err := builder.Build(ctx, "sandwich", "")
func NewBuilder(ctx context.Context) (manglekit.ProgrammaticBuilder, error) {
	registry := manglekit.NewRegistry()
	all.Register(registry)

	// Initialize genkit with provider plugins to support all LLM components
	// For programmatic use cases where LLM providers are often used, we pre-register plugins
	g := manglekit.InitGenkitWithProviders(ctx)

	logger := logger.NewStdLogger()
	obs := core.Observability{
		Logger: logger,
	}

	b, err := manglekit.NewBuilder(ctx, registry, obs, g)
	if err != nil {
		return nil, fmt.Errorf("failed to create new builder: %w", err)
	}
	return b, nil
}
