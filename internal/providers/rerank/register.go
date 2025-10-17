package rerank

import (
	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/internal/providers/rerank/cosine"
	"github.com/duynguyendang/manglekit/sdk"
)

func init() {
	Register(sdk.GlobalRegistry())
}

// Register registers all reranker providers.
func Register(r *manglekit.Registry) {
	cosine.Register(r)
}