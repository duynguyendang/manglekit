package sdk

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/config"
	"github.com/duynguyendang/manglekit/core"
)

// FromConfig loads a Manglekit orchestrator from a YAML configuration byte slice.
// The caller is responsible for creating and populating the registry.
func FromConfig(ctx context.Context, registry *manglekit.Registry, data []byte) (core.Orchestrator, error) {
	if registry == nil {
		return nil, fmt.Errorf("registry cannot be nil")
	}

	cfg, err := config.ParseConfig(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	builder := manglekit.NewBuilder(registry)
	for _, comp := range cfg.Components {
		// Note: This uses WithKind, which is a code smell. A future refactor
		// could improve this by using the registry to decode the params into
		// a typed options struct and using With().
		builder.WithKind(comp.Kind, comp.Name, comp.Params)
	}

	orch, _, err := builder.Build(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to build orchestrator: %w", err)
	}
	return orch, nil
}
