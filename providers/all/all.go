//go:build !testhooks

package all

import (
	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/internal/embedders"
	"github.com/duynguyendang/manglekit/internal/providers/retrievers/bm25"
	"github.com/duynguyendang/manglekit/internal/providers/retrievers/dense"
	"github.com/duynguyendang/manglekit/internal/providers/retrievers/hybrid"
	"github.com/duynguyendang/manglekit/internal/providers/llm"
	"github.com/duynguyendang/manglekit/internal/providers/orchestrators"
	"github.com/duynguyendang/manglekit/internal/providers/rerank"
	"github.com/duynguyendang/manglekit/internal/providers/rerank/cosine"
	"github.com/duynguyendang/manglekit/internal/providers/retrievers"
	"github.com/duynguyendang/manglekit/internal/providers/rules"
	"github.com/duynguyendang/manglekit/internal/providers/rules/mangle"
	"github.com/duynguyendang/manglekit/internal/providers/schemaparsers"
	"github.com/duynguyendang/manglekit/internal/providers/state"
	"github.com/duynguyendang/manglekit/internal/providers/state/inmemory"
	"github.com/duynguyendang/manglekit/internal/vectorstores"
	"github.com/duynguyendang/manglekit/pipeline/declarative"
	"github.com/duynguyendang/manglekit/pipeline/sandwich"
)

func Register(r *manglekit.Registry) {
	// Provider Factories and Options
	bm25.Register(r)
	cosine.Register(r)
	declarative.Register(r)
	dense.Register(r)
	hybrid.Register(r)
	inmemory.Register(r)
	mangle.Register(r)
	sandwich.Register(r)

	// NEW: Aggregate Registrations
	llm.Register(r)
	embedders.Register(r)
	schemaparsers.Register(r)

	// Component Handlers
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
	r.RegisterHandler(vectorstores.NewHandler())
}
