package main

import (
	"context"
	"flag"
	"log"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/config"
	_ "github.com/duynguyendang/manglekit/providers/all" // Import to register all providers
	"github.com/duynguyendang/manglekit/sdk"
)

func main() {
	configFile := flag.String("config", "config.yaml", "Path to the configuration file")
	flag.Parse()

	// The global registry is populated by the `providers/all` import.
	registry := sdk.GlobalRegistry()

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
