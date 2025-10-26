package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/sdk"
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

	// 1. Load the orchestrator from the configuration.
	orch, err := sdk.Load(ctx, configData)
	if err != nil {
		log.Fatalf("failed to load orchestrator: %v", err)
	}
	defer orch.Close(ctx)

	// 2. Execute the query.
	answer, err := orch.Execute(ctx, "session-123", core.Query{Text: query})
	if err != nil {
		log.Fatalf("failed to execute query: %v", err)
	}

	fmt.Println(answer.Text)
}
