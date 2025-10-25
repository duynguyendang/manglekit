package vectorstores

import (
	"github.com/duynguyendang/manglekit/core"
)

// NewHandler returns a new ComponentHandler for the vectorstore kind.
func NewHandler() core.ComponentHandler {
	return &Handler{}
}
