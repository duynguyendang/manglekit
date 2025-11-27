//go:build testhooks

package all

import (
	"github.com/duynguyendang/manglekit/v1"
	"github.com/duynguyendang/manglekit/v1/internal/providers/llm"
	"github.com/duynguyendang/manglekit/v1/internal/providers/rerank/cosine"
	"github.com/duynguyendang/manglekit/v1/internal/providers/retrievers/bm25"
	"github.com/duynguyendang/manglekit/v1/internal/providers/retrievers/hybrid"
	"github.com/duynguyendang/manglekit/v1/internal/providers/schemaparsers/jsonschema"
	"github.com/duynguyendang/manglekit/v1/internal/providers/schemaparsers/rdf"
	"github.com/duynguyendang/manglekit/v1/internal/providers/state/inmemory"
	"github.com/duynguyendang/manglekit/v1/pipeline/declarative"
	"github.com/duynguyendang/manglekit/v1/pipeline/sandwich"
)

func Register(r *manglekit.Registry) {
	bm25.Register(r)
	cosine.Register(r)
	declarative.Register(r)
	hybrid.Register(r)
	inmemory.Register(r)
	jsonschema.Register(r)
	llm.RegisterGoogle(r)
	llm.RegisterOpenAI(r)
	rdf.Register(r)
	sandwich.Register(r)
}
