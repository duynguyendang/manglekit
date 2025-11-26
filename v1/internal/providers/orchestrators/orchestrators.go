package orchestrators

import (
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/pipeline/declarative"
	"github.com/duynguyendang/manglekit/pipeline/sandwich"
)

// Handlers returns a slice of orchestrator handlers
func Handlers() []core.ComponentHandler {
	return []core.ComponentHandler{
		declarative.NewHandler(),
		sandwich.NewHandler(),
	}
}
