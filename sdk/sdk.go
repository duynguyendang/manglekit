package sdk

import (
	"context"
	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/providers/all"
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
