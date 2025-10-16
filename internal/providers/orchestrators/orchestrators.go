package orchestrators

import (
	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/pipeline"
)

func Register(r *manglekit.Registry) {
	r.RegisterOrchestrator("sandwich", func(opts core.Options) (core.Orchestrator, error) {
		return pipeline.NewSandwich(opts)
	})
}