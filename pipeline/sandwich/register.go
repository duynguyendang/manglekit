package sandwich

import (
	"context"
	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
)

// Register registers the sandwich orchestrator with the MangleKit registry.
func Register(r *manglekit.Registry) {
	manglekit.Register(r, &Options{},
		func(ctx context.Context, deps diapi.SandwichDeps, cfg *Options) (core.Orchestrator, error) {
			return NewFactory(cfg).Build(ctx, deps)
		},
	)
	r.RegisterHandler(NewHandler())
}
