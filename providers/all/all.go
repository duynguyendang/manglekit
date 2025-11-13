//go:build !testhooks

package all

import (
	"errors"
	"fmt"
	"log"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/internal/embedders"
	"github.com/duynguyendang/manglekit/internal/providers/llm"
	"github.com/duynguyendang/manglekit/internal/providers/orchestrators"
	"github.com/duynguyendang/manglekit/internal/providers/planners"
	"github.com/duynguyendang/manglekit/internal/providers/reasoners"
	"github.com/duynguyendang/manglekit/internal/providers/rerank"
	"github.com/duynguyendang/manglekit/internal/providers/rerank/cosine"
	"github.com/duynguyendang/manglekit/internal/providers/retrievers"
	"github.com/duynguyendang/manglekit/internal/providers/retrievers/bm25"
	"github.com/duynguyendang/manglekit/internal/providers/retrievers/genkitretriever"
	"github.com/duynguyendang/manglekit/internal/providers/retrievers/hybrid"
	"github.com/duynguyendang/manglekit/internal/providers/rules"
	"github.com/duynguyendang/manglekit/internal/providers/rules/mangle"
	"github.com/duynguyendang/manglekit/internal/providers/schemaparsers"
	"github.com/duynguyendang/manglekit/internal/providers/state"
	"github.com/duynguyendang/manglekit/internal/providers/state/inmemory"
	"github.com/duynguyendang/manglekit/internal/providers/tools"
	httpTool "github.com/duynguyendang/manglekit/internal/providers/tools/http"
	"github.com/duynguyendang/manglekit/pipeline/declarative"
	"github.com/duynguyendang/manglekit/pipeline/sandwich"
)

func Register(r *manglekit.Registry) {
	var errs []error

	// Provider Factories and Options
	bm25.Register(r)
	cosine.Register(r)
	declarative.Register(r)
	if err := genkitretriever.Register(r); err != nil {
		errs = append(errs, fmt.Errorf("genkitretriever registration: %w", err))
	}
	hybrid.Register(r)
	inmemory.Register(r)
	mangle.Register(r)
	sandwich.Register(r)
	httpTool.Register(r)

	// NEW: Aggregate Registrations with error handling
	llm.Register(r)
	if err := embedders.Register(r); err != nil {
		errs = append(errs, fmt.Errorf("embedders registration: %w", err))
	}
	if err := schemaparsers.Register(r); err != nil {
		errs = append(errs, fmt.Errorf("schemaparsers registration: %w", err))
	}
	reasoners.Register(r)

	// If there were any registration errors, log them
	if len(errs) > 0 {
		combined := errors.Join(errs...)
		log.Printf("WARNING: Some providers failed to register: %v\n", combined)
	}

	// Component Handlers
	tools.Register(r)
	r.RegisterHandler(retrievers.NewHandler())
	r.RegisterHandler(llm.NewHandler())
	r.RegisterHandler(embedders.NewHandler())
	r.RegisterHandler(rerank.NewHandler())
	r.RegisterHandler(rules.NewHandler())
	r.RegisterHandler(state.NewHandler())
	r.RegisterHandler(schemaparsers.NewHandler())
	for _, h := range orchestrators.Handlers() {
		r.RegisterHandler(h)
	}
	r.RegisterHandler(planners.NewHandler())
}
