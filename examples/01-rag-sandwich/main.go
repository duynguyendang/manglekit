package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/sdk"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: go run ./examples/01-rag-sandwich \"<query>\"")
	}
	prompt := strings.Join(os.Args[1:], " ")
	ctx := context.Background()

	// 1. Read the configuration file.
	configData, err := os.ReadFile("examples/01-rag-sandwich/config.yaml")
	if err != nil {
		log.Fatalf("Error reading config file: %v", err)
	}

	// 2. Load the orchestrator from the configuration.
	orch, err := sdk.Load(ctx, configData)
	if err != nil {
		log.Fatalf("Error building orchestrator: %v", err)
	}
	defer orch.Close(ctx) // 3. Ensure graceful shutdown

	// 4. Execute the orchestrator
	fmt.Println("--- Manglekit Example 01: RAG Sandwich ---")
	fmt.Printf("Query: %s\n", prompt)

	ans, err := orch.Execute(ctx, "session-id", core.Query{Text: prompt})
	if err != nil {
		log.Fatalf("Error executing pipeline: %v", err)
	}

	fmt.Printf("\nAnswer: %s\n", ans.Text)
}
