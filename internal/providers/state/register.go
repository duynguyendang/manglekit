package state

import (
	"github.com/duynguyendang/manglekit/core"
)

// NewHandler returns a new ComponentHandler for the state kind.
func NewHandler() core.ComponentHandler {
	return &Handler{}
}
