package orchestrators

import (
	"github.com/duynguyendang/manglekit/v1/core"
	"github.com/duynguyendang/manglekit/v1/pipeline/declarative"
	"github.com/duynguyendang/manglekit/v1/pipeline/sandwich"
)

// Handlers returns a slice of orchestrator handlers
func Handlers() []core.ComponentHandler {
	return []core.ComponentHandler{
		declarative.NewHandler(),
		sandwich.NewHandler(),
	}
}
