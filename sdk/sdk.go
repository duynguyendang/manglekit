package sdk

import (
	"context"
	"fmt"
	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/logger"
	"github.com/duynguyendang/manglekit/providers/all"
	"github.com/firebase/genkit/go/genkit"
)

// Load is a high-level function that loads a Manglekit orchestrator from a YAML
// configuration byte slice. It handles registry creation and component handler
// registration automatically.
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
func NewBuilder(ctx context.Context) (manglekit.ProgrammaticBuilder, error) {
	registry := manglekit.NewRegistry()
	all.Register(registry)

	g := genkit.Init(ctx)

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
