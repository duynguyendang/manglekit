package schemaparsers

import (
	"github.com/duynguyendang/manglekit/core"
)

// NewHandler returns a new ComponentHandler for the schemaparser kind.
func NewHandler() core.ComponentHandler {
	return &Handler{}
}
