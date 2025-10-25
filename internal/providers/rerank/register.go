package rerank

import (
	"github.com/duynguyendang/manglekit/core"
)

// NewHandler returns a new ComponentHandler for the rerank kind.
func NewHandler() core.ComponentHandler {
	return &Handler{}
}
