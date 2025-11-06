//go:build testhooks

package all

import (
	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/internal/providers/retrievers/bm25"
	"github.com/duynguyendang/manglekit/internal/providers/retrievers/dense"
	"github.com/duynguyendang/manglekit/internal/providers/retrievers/hybrid"
	"github.com/duynguyendang/manglekit/internal/providers/llm"
	"github.com/duynguyendang/manglekit/internal/providers/rerank/cosine"
	"github.com/duynguyendang/manglekit/internal/providers/schemaparsers/jsonschema"
	"github.com/duynguyendang/manglekit/internal/providers/schemaparsers/rdf"
	"github.com/duynguyendang/manglekit/internal/providers/state/inmemory"
	"github.com/duynguyendang/manglekit/pipeline/declarative"
	"github.com/duynguyendang/manglekit/pipeline/sandwich"
)

func Register(r *manglekit.Registry) {
	bm25.Register(r)
	cosine.Register(r)
	declarative.Register(r)
	dense.Register(r)
	hybrid.Register(r)
	inmemory.Register(r)
	jsonschema.Register(r)
	llm.RegisterGoogle(r)
	llm.RegisterOpenAI(r)
	rdf.Register(r)
	sandwich.Register(r)
}
