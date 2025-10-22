package retrievers

import (
	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/internal/providers/bm25"
	"github.com/duynguyendang/manglekit/internal/providers/dense"
	"github.com/duynguyendang/manglekit/internal/providers/hybrid"
	"github.com/duynguyendang/manglekit/internal/providers/retrievers/inmemory"
)

// Register registers all retriever providers and the retriever kind handler.
func Register(r *manglekit.Registry) {
	inmemory.Register(r)
	bm25.Register(r)
	dense.Register(r)
	hybrid.Register(r)
	r.RegisterHandler(&Handler{})
}
