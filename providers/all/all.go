// Package all provides a function to register all standard Manglekit providers.
package all

import (
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/embedders"
	"github.com/duynguyendang/manglekit/internal/providers/llm"
	"github.com/duynguyendang/manglekit/internal/providers/orchestrators"
	"github.com/duynguyendang/manglekit/internal/providers/rerank"
	"github.com/duynguyendang/manglekit/internal/providers/retrievers"
	"github.com/duynguyendang/manglekit/internal/providers/rules"
	"github.com/duynguyendang/manglekit/internal/providers/schemaparsers"
	"github.com/duynguyendang/manglekit/internal/providers/state"
	"github.com/duynguyendang/manglekit/internal/vectorstores"
)

// ComponentHandlers collects ALL handlers for the Builder
func ComponentHandlers() []core.ComponentHandler {
	// Start with standard component handlers
	handlers := []core.ComponentHandler{
		retrievers.NewHandler(),
		llm.NewHandler(),
		rerank.NewHandler(),
		rules.NewHandler(),
		schemaparsers.NewHandler(),
		state.NewHandler(),
		embedders.NewHandler(),
		vectorstores.NewHandler(),
	}

	// Add the orchestrator handlers to the list
	handlers = append(handlers, orchestrators.Handlers()...)

	return handlers
}
