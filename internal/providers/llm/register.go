package llm

import (
	"github.com/duynguyendang/manglekit/core"
)

// NewHandler returns a new ComponentHandler for the llm kind.
func NewHandler() core.ComponentHandler {
	return &Handler{}
}
