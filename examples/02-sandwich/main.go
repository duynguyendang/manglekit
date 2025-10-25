package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/providers/llm"
	"github.com/duynguyendang/manglekit/internal/providers/mock"
	"github.com/duynguyendang/manglekit/internal/providers/orchestrators"
	"github.com/duynguyendang/manglekit/internal/providers/retrievers"
	"github.com/duynguyendang/manglekit/internal/providers/state"
	"github.com/duynguyendang/manglekit/internal/providers/state/inmemory"
	"github.com/duynguyendang/manglekit/pipeline/sandwich"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s <query>", os.Args[0])
	}
	query := os.Args[1]
	ctx := context.Background()

	configData, err := os.ReadFile("examples/02-sandwich/config.yaml")
	if err != nil {
		log.Fatalf("failed to read config file: %v", err)
	}

	// 1. Create a registry and register all the provider factories and their options types.
	registry := manglekit.NewRegistry()
	mock.Register(registry)
	inmemory.Register(registry)
	manglekit.Register(registry, &sandwich.SandwichOptions{}, sandwich.NewSandwich)


	// 2. Create a builder and give it the handlers for each component kind.
	builder := manglekit.NewBuilder(registry).WithHandlers(
		llm.NewHandler(),
		retrievers.NewHandler(),
		state.NewHandler(),
	).WithHandlers(orchestrators.Handlers()...)

	// 3. Build the orchestrator from the YAML config.
	orch, _, err := builder.FromConfig(ctx, configData)
	if err != nil {
		log.Fatalf("failed to load orchestrator: %v", err)
	}
	defer orch.Close(ctx)

	// 4. Execute the query.
	answer, err := orch.Execute(ctx, "session-123", core.Query{Text: query})
	if err != nil {
		log.Fatalf("failed to execute query: %v", err)
	}

	fmt.Println(answer.Text)
}
