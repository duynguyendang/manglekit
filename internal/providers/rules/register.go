package rules

import (
	"github.com/duynguyendang/manglekit/core"
)

// NewHandler returns a new ComponentHandler for the rules kind.
func NewHandler() core.ComponentHandler {
	return &Handler{}
}
