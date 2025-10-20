package rerank

import (
	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/internal/providers/rerank/cosine"
)

// Register registers all reranker providers.
func Register(r *manglekit.Registry) {
	cosine.Register(r)
}
