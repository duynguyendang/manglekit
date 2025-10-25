package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/config"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/providers/all"
	"gopkg.in/yaml.v3"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: go run ./examples/01-rag-sandwich \"<query>\"")
	}
	prompt := strings.Join(os.Args[1:], " ")
	ctx := context.Background()

	// 1. Create a new registry.
	registry := manglekit.NewRegistry()

	// 2. Read the configuration file.
	configData, err := os.ReadFile("examples/01-rag-sandwich/config.yaml")
	if err != nil {
		log.Fatalf("Error reading config file: %v", err)
	}

	// 3. Parse the config to get the orchestrator name.
	var cfg config.Config
	if err := yaml.Unmarshal(configData, &cfg); err != nil {
		log.Fatalf("Error unmarshalling config: %v", err)
	}

	// 4. Create a new builder and load from config.
	builder := manglekit.NewBuilder(registry).WithHandlers(all.ComponentHandlers()...)
	orch, _, err := builder.FromConfig(ctx, configData)
	if err != nil {
		log.Fatalf("Error building orchestrator: %v", err)
	}
	defer orch.Close(ctx) // 5. Ensure graceful shutdown

	// 6. Execute the orchestrator
	fmt.Println("--- Manglekit Example 01: RAG Sandwich ---")
	fmt.Printf("Query: %s\n", prompt)

	ans, err := orch.Execute(ctx, "session-id", core.Query{Text: prompt})
	if err != nil {
		log.Fatalf("Error executing pipeline: %v", err)
	}

	fmt.Printf("\nAnswer: %s\n", ans.Text)
}
