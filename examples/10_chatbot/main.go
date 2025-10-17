package main

import (
	"context"
	"flag"
	"log"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/config"
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
)

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

	builder, err := manglekit.NewBuilderFromConfig(context.Background(), cfg, registry, nil)
	if err != nil {
		log.Fatalf("failed to create builder: %v", err)
	}
	ctx := context.Background()
	orch, _, err := builder.Build(ctx)
	if err != nil {
		log.Fatalf("failed to build orchestrator: %v", err)
	}
	defer orch.Close(ctx)

	// Now you can use the orchestrator to execute queries.
	// For example:
	// answer, err := orch.Execute(ctx, "session-123", core.Query{Text: "What is Manglekit?"})
	// if err != nil {
	// 	log.Fatalf("failed to execute query: %v", err)
	// }
	// log.Printf("Answer: %s", answer.Text)
}
