package orchestrators

import (
	"context"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/pipeline"
	"github.com/duynguyendang/manglekit/pipeline/declarative"
)

func Register(r *manglekit.Registry) {
	// Register the "sandwich" orchestrator.
	// It has no specific options of its own, so we use `core.NilOptions`.
	// Its dependencies are provided by the `core.Resolved` struct, which is
	// passed by the builder at construction time.
	manglekit.Register[core.Orchestrator, core.Resolved, core.NilOptions](
		r,
		core.NilOptions{Name: "sandwich", Kind: core.KindOrchestrator},
		func(ctx context.Context, deps core.Resolved, _ core.NilOptions) (core.Orchestrator, error) {
			return pipeline.NewSandwich(ctx, deps)
		},
	)

	// Register the "declarative" orchestrator.
	// This orchestrator is configured with a list of tool steps.
	manglekit.Register[core.Orchestrator, core.Resolved, declarative.Options](
		r,
		declarative.Options{},
		declarative.NewDeclarative,
	)
}