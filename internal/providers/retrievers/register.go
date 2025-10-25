package retrievers

import (
	"github.com/duynguyendang/manglekit/core"
)

// NewHandler returns a new ComponentHandler for the retriever kind.
func NewHandler() core.ComponentHandler {
	return &Handler{}
}
