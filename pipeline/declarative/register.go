package declarative

import (
	"context"
	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
)

// Register registers the declarative orchestrator with the MangleKit registry.
func Register(r *manglekit.Registry) {
	manglekit.Register(r, &Options{},
		func(ctx context.Context, deps diapi.NoopDeps, cfg *Options) (core.Orchestrator, error) {
			// The declarative orchestrator is a bit different, it needs the full resolved deps.
			// This is a temporary workaround until the DI system is more flexible.
			return nil, nil
		},
	)
	r.RegisterHandler(NewHandler())
}
