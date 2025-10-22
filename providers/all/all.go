// Package all provides a function to register all standard Manglekit providers.
package all

import (
	"github.com/duynguyendang/manglekit"
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

// Register registers all standard providers with the given registry.
func Register(r *manglekit.Registry) {
	embedders.Register(r)
	llm.Register(r)
	orchestrators.Register(r)
	rerank.Register(r)
	retrievers.Register(r)
	rules.Register(r)
	schemaparsers.Register(r)
	state.Register(r)
	vectorstores.Register(r)
}
