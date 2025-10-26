package sdk

import (
	"context"
	"fmt"
	"github.com/duynguyendang/manglekit/internal/logger"
	"github.com/firebase/genkit/go/genkit"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/providers/all"
)

// Load is a high-level function that loads a Manglekit orchestrator from a YAML
// configuration byte slice. It handles registry creation and component handler
// registration automatically.
func Load(ctx context.Context, data []byte) (core.Orchestrator, error) {
	// 1. Create a new registry.
	registry := manglekit.NewRegistry()

	l := logger.NewStdLogger()
	obs := core.Observability{Logger: l}
	g := genkit.Init(ctx)

	// 2. Create a new builder and register all component handlers.
	builder, err := manglekit.NewBuilder(ctx, registry, obs, g)
	if err != nil {
		return nil, fmt.Errorf("failed to create new builder: %w", err)
	}
	builder.WithHandlers(all.ComponentHandlers()...)

	// 3. Build the orchestrator from the configuration.
	orch, _, err := builder.FromConfig(ctx, data)
	if err != nil {
		return nil, fmt.Errorf("failed to build orchestrator from config: %w", err)
	}
	return orch, nil
}

// FromConfig loads a Manglekit orchestrator from a YAML configuration byte slice.
// The caller is responsible for creating and populating the registry.
func FromConfig(ctx context.Context, registry *manglekit.Registry, data []byte) (core.Orchestrator, error) {
	if registry == nil {
		return nil, fmt.Errorf("registry cannot be nil")
	}

	l := logger.NewStdLogger()
	obs := core.Observability{Logger: l}
	g := genkit.Init(ctx)

	builder, err := manglekit.NewBuilder(ctx, registry, obs, g)
	if err != nil {
		return nil, fmt.Errorf("failed to create new builder: %w", err)
	}

	// 3. Build the orchestrator from the configuration.
	orch, _, err := builder.FromConfig(ctx, data)
	if err != nil {
		return nil, fmt.Errorf("failed to build orchestrator from config: %w", err)
	}
	return orch, nil
}
