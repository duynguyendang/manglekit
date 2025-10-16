package main

import (
	"context"
	"flag"
	"io"
	"log"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/config"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
	"github.com/duynguyendang/manglekit/internal/embedders/google"
	"github.com/duynguyendang/manglekit/internal/embedders/openai"
	"github.com/duynguyendang/manglekit/internal/providers/bm25"
	"github.com/duynguyendang/manglekit/internal/providers/dense"
	"github.com/duynguyendang/manglekit/internal/providers/hybrid"
	"github.com/duynguyendang/manglekit/internal/providers/llm"
	"github.com/duynguyendang/manglekit/internal/providers/mangle"
	"github.com/duynguyendang/manglekit/internal/providers/rerank/cosine"
	inmemory "github.com/duynguyendang/manglekit/internal/providers/retrievers/inmemory"
	"github.com/duynguyendang/manglekit/internal/providers/schemaparsers/jsonschema"
	"github.com/duynguyendang/manglekit/internal/providers/schemaparsers/rdf"
	"github.com/duynguyendang/manglekit/retrieve"
	"github.com/google/mangle/ast"
)

type mockSchemaParser struct{}

func (p *mockSchemaParser) Predicates() []ast.PredicateSym {
	return []ast.PredicateSym{
		{Symbol: "schema", Arity: 1},
		{Symbol: "field", Arity: 3},
	}
}

func (p *mockSchemaParser) Parse(source io.Reader) ([]ast.Atom, error) {
	return []ast.Atom{
		ast.NewAtom("schema", ast.String("mock-schema")),
		ast.NewAtom("field", ast.String("mock-schema"), ast.String("field1"), ast.String("string")),
	}, nil
}

func registerAllProviders(r *manglekit.Registry) {
	// LLM Providers
	llm.RegisterGoogle(r)
	llm.RegisterOpenAI(r)

	// Embedder Providers
	google.Register(r)
	openai.Register(r)

	// Retriever Providers
	inmemory.Register(r)
	bm25.Register(r)
	dense.Register(r)
	hybrid.Register(r)

	// Reranker Providers
	cosine.Register(r)

	// Rules Providers
	mangle.Register(r)

	// Schema Parser Providers
	jsonschema.Register(r)
	rdf.Register(r)
	r.RegisterSchemaParser("mock-schema-parser", func(ctx context.Context, deps diapi.NoopDeps, cfg any) (core.SchemaParser, error) {
		return &mockSchemaParser{}, nil
	})

	// Options
	r.RegisterOptions("bm25", (*retrieve.BM25Options)(nil))
}

func main() {
	configFile := flag.String("config", "config.yaml", "Path to the configuration file")
	flag.Parse()

	registry := manglekit.NewRegistry()
	registerAllProviders(registry)

	cfg, err := config.LoadFromYAMLFile(*configFile)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	builder, err := manglekit.NewBuilderFromConfig(context.Background(), cfg, registry)
	if err != nil {
		log.Fatalf("failed to create builder: %v", err)
	}
	ctx := context.Background()
	orch, _, err := builder.Build(ctx)
	if err != nil {
		log.Fatalf("failed to build orchestrator: %v", err)
	}
	defer orch.Close(ctx)
}
