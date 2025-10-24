package rules

import (
	"context"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
	"github.com/duynguyendang/manglekit/internal/providers/mangle"
)

// Register registers all rules providers and the rules kind handler.
func Register(r *manglekit.Registry) {
	must := func(err error) {
		if err != nil {
			panic(err)
		}
	}

	must(manglekit.Register(r, core.MangleOptions{},
		func(ctx context.Context, deps diapi.RuleSetDeps, cfg core.MangleOptions) (core.RuleSet, error) {
			return mangle.New(ctx, cfg, r)
		},
	))
	r.RegisterHandler(&Handler{})
}
