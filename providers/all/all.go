// Package all provides a function to register all standard Manglekit providers.
package all

import (
	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/internal/providers/bm25"
	"github.com/duynguyendang/manglekit/internal/providers/dense"
	"github.com/duynguyendang/manglekit/internal/providers/hybrid"
	"github.com/duynguyendang/manglekit/internal/providers/llm"
	"github.com/duynguyendang/manglekit/internal/providers/rerank/cosine"
	"github.com/duynguyendang/manglekit/internal/providers/state/inmemory"
	"github.com/duynguyendang/manglekit/pipeline/declarative"
	"github.com/duynguyendang/manglekit/pipeline/sandwich"
)

// Register registers all standard providers with the MangleKit registry.
func Register(r *manglekit.Registry) {
	bm25.Register(r)
	dense.Register(r)
	hybrid.Register(r)
	llm.Register(r)
	cosine.Register(r)
	inmemory.Register(r)
	declarative.Register(r)
	sandwich.Register(r)
}
