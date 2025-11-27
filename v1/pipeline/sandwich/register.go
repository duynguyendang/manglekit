package sandwich

import (
	"context"
	"fmt"
	"github.com/duynguyendang/manglekit/v1"
	"github.com/duynguyendang/manglekit/v1/core"
	"github.com/duynguyendang/manglekit/v1/core/diapi"
)

// Register registers the sandwich orchestrator with the MangleKit registry.
func Register(r *manglekit.Registry) {
	manglekit.Register(r, &Options{},
		func(ctx context.Context, deps diapi.SandwichDeps, cfg *Options) (core.Orchestrator, error) {
			factory := NewFactory()
			built, err := factory.Build(ctx, deps, cfg)
			if err != nil {
				return nil, fmt.Errorf("failed to build sandwich orchestrator: %w", err)
			}
			orchestrator, ok := built.(core.Orchestrator)
			if !ok {
				return nil, fmt.Errorf("built component is not a core.Orchestrator, but %T", built)
			}
			return orchestrator, nil
		},
	)
	r.RegisterHandler(NewHandler())
}
