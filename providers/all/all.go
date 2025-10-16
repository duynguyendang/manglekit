// Package all provides a convenient way to register all standard Manglekit providers.
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
	"github.com/duynguyendang/manglekit/internal/providers/mock"
	"github.com/duynguyendang/manglekit/internal/providers/rerank/cosine"
	"github.com/duynguyendang/manglekit/internal/providers/retrievers/inmemory"
	"github.com/duynguyendang/manglekit/internal/providers/schemaparsers/jsonschema"
	"github.com/duynguyendang/manglekit/internal/providers/schemaparsers/rdf"
	inmemorystate "github.com/duynguyendang/manglekit/internal/providers/state/inmemory"
	"github.com/duynguyendang/manglekit/internal/providers/orchestrators"
	"github.com/duynguyendang/manglekit/internal/providers/state/redis"
	"github.com/duynguyendang/manglekit/providers"
)

// Standard is a provider set that includes all standard Manglekit providers.
var Standard = providers.NewSet().
	With(orchestrators.Register).
	With(llm.Register).
	With(google.Register).
	With(openai.Register).
	With(bm25.Register).
	With(dense.Register).
	With(hybrid.Register).
	With(inmemory.Register).
	With(cosine.Register).
	With(mangle.Register).
	With(jsonschema.Register).
	With(rdf.Register).
	With(inmemorystate.Register).
	With(redis.Register).
	With(mock.Register)

// RegisterAll applies all standard provider registrations to the given registry.
func RegisterAll(r *manglekit.Registry) {
	Standard.ApplyTo(r)
}