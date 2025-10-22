package rerank

import (
	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/internal/providers/rerank/cosine"
)

// Register registers all reranker providers and the reranker kind handler.
func Register(r *manglekit.Registry) {
	cosine.Register(r)
	r.RegisterHandler(&Handler{})
}
