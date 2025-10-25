package embedders

import (
	"github.com/duynguyendang/manglekit/core"
)

// NewHandler returns a new ComponentHandler for the embedder kind.
func NewHandler() core.ComponentHandler {
	return &Handler{}
}
