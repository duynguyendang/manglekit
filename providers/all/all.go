// Package all provides a function to register all standard Manglekit providers.
package all

import (
	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/internal/embedders/google"
	"github.com/duynguyendang/manglekit/internal/embedders/openai"
	"github.com/duynguyendang/manglekit/internal/providers/bm25"
	"github.com/duynguyendang/manglekit/internal/providers/dense"
	"github.com/duynguyendang/manglekit/internal/providers/hybrid"
	"github.com/duynguyendang/manglekit/internal/providers/llm"
	"github.com/duynguyendang/manglekit/internal/providers/mangle"
	"github.com/duynguyendang/manglekit/internal/providers/orchestrators"
	"github.com/duynguyendang/manglekit/internal/providers/rerank"
	"github.com/duynguyendang/manglekit/internal/providers/retrievers"
	"github.com/duynguyendang/manglekit/internal/providers/schemaparsers/jsonschema"
	"github.com/duynguyendang/manglekit/internal/providers/schemaparsers/rdf"
	"github.com/duynguyendang/manglekit/internal/providers/state"
	"github.com/duynguyendang/manglekit/internal/vectorstores"
)

// Register registers all standard providers with the given registry.
func Register(r *manglekit.Registry) {
	google.Register(r)
	openai.Register(r)
	bm25.Register(r)
	dense.Register(r)
	hybrid.Register(r)
	llm.Register(r)
	mangle.Register(r)
	orchestrators.Register(r)
	rerank.Register(r)
	retrievers.Register(r)
	jsonschema.Register(r)
	rdf.Register(r)
	state.Register(r)
	vectorstores.Register(r)
}
