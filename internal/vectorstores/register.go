package vectorstores

import (
	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/internal/vectorstores/localvec"
	"github.com/duynguyendang/manglekit/sdk"
)

func init() {
	Register(sdk.GlobalRegistry())
}

// Register registers all vector store providers.
func Register(r *manglekit.Registry) {
	localvec.Register(r)
}