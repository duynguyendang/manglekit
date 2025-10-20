package orchestrators

import (
	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/pipeline"
	"github.com/duynguyendang/manglekit/pipeline/declarative"
)

func Register(r *manglekit.Registry) {
	// Register the "sandwich" orchestrator.
	// Register the "sandwich" orchestrator.
	// It is configured with `pipeline.SandwichOptions` to specify its core components.
	manglekit.Register[core.Orchestrator, core.Resolved, pipeline.SandwichOptions](
		r,
		pipeline.SandwichOptions{},
		pipeline.NewSandwich,
	)

	// Register the "declarative" orchestrator.
	// This orchestrator is configured with a list of tool steps.
	manglekit.Register[core.Orchestrator, core.Resolved, declarative.Options](
		r,
		declarative.Options{},
		declarative.NewDeclarative,
	)
}