package retrievers

import (
	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/internal/providers/bm25"
	"github.com/duynguyendang/manglekit/internal/providers/dense"
	"github.com/duynguyendang/manglekit/internal/providers/hybrid"
	"github.com/duynguyendang/manglekit/internal/providers/retrievers/inmemory"
	"github.com/duynguyendang/manglekit/sdk"
)

func init() {
	Register(sdk.GlobalRegistry())
}

// Register registers all retriever providers.
func Register(r *manglekit.Registry) {
	inmemory.Register(r)
	bm25.Register(r)
	dense.Register(r)
	hybrid.Register(r)
}