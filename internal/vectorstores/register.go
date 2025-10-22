package vectorstores

import (
	"context"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
	"github.com/duynguyendang/manglekit/internal/vectorstores/localvec"
)

// Register registers all vector store providers and the vector store kind handler.
func Register(r *manglekit.Registry) {
	manglekit.Register(r, core.LocalvecOptions{},
		func(ctx context.Context, deps diapi.VectorStoreDeps, cfg core.LocalvecOptions) (core.VectorStore, error) {
			return localvec.New(ctx, cfg, deps.Embedder)
		},
	)
	r.RegisterHandler(&Handler{})
}
