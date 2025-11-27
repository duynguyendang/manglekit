package declarative

import (
	"context"
	"github.com/duynguyendang/manglekit/v1"
	"github.com/duynguyendang/manglekit/v1/core"
	"github.com/duynguyendang/manglekit/v1/core/diapi"
)

// Register registers the declarative orchestrator with the MangleKit registry.
func Register(r *manglekit.Registry) {
	manglekit.Register(r, &Options{},
		func(ctx context.Context, deps diapi.DeclarativeOrchestratorDeps, cfg *Options) (core.Orchestrator, error) {
			return NewDeclarative(ctx, deps, cfg)
		},
	)
}
