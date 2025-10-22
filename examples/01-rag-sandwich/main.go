package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/providers/all"
	"github.com/duynguyendang/manglekit/sdk"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: go run ./examples/01-rag-sandwich \"<query>\"")
	}
	prompt := strings.Join(os.Args[1:], " ")
	ctx := context.Background()

	// 1. Create a new registry and register all providers.
	registry := manglekit.NewRegistry()
	all.Register(registry)

	// 2. Read the configuration file.
	configData, err := os.ReadFile("examples/01-rag-sandwich/config.yaml")
	if err != nil {
		log.Fatalf("Error reading config file: %v", err)
	}

	// 3. Load pipeline from config
	orch, err := sdk.FromConfig(ctx, registry, configData)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}
	defer orch.Close(ctx) // 4. Ensure graceful shutdown

	// 5. Execute the orchestrator
	fmt.Println("--- Manglekit Example 01: RAG Sandwich ---")
	fmt.Printf("Query: %s\n", prompt)

	ans, err := orch.Execute(ctx, "session-id", core.Query{Text: prompt})
	if err != nil {
		log.Fatalf("Error executing pipeline: %v", err)
	}

	fmt.Printf("\nAnswer: %s\n", ans.Text)
}
