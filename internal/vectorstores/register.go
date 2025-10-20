package vectorstores

import (
	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/internal/vectorstores/localvec"
)

// Register registers all vector store providers.
func Register(r *manglekit.Registry) {
	localvec.Register(r)
}
