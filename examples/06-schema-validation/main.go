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
	_ "github.com/duynguyendang/manglekit/providers/all" // Import to register all providers
	"github.com/duynguyendang/manglekit/sdk"
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

type mockSchemaParserOptions struct{}

func (o mockSchemaParserOptions) ProviderName() string { return "mock-schema-parser" }
func (o mockSchemaParserOptions) ProviderKind() core.Kind   { return core.KindSchemaParser }

func main() {
	configFile := flag.String("config", "config.yaml", "Path to the configuration file")
	flag.Parse()

	registry := sdk.GlobalRegistry()
	manglekit.Register(registry, mockSchemaParserOptions{},
		func(ctx context.Context, deps diapi.NoopDeps, cfg mockSchemaParserOptions) (core.SchemaParser, error) {
			return &mockSchemaParser{}, nil
		},
	)

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
}
