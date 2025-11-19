// Package main demonstrates how to use planners in Manglekit.
// This example shows configuration-based setup of a symbolic planner orchestrator.
package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/registry"
	_ "github.com/duynguyendang/manglekit/providers/all" // Auto-registers all standard providers
	"github.com/duynguyendang/manglekit/sdk"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Note: .env file not found, will fall back to environment variables")
	}

	ctx := context.Background()

	// This example demonstrates using a symbolic planner in Manglekit.
	// The planner generates multi-step execution plans based on reasoner outputs.

	fmt.Println("=== Symbolic Planner Example ===")
	fmt.Println()

	// Try to load a config file with a planner-based orchestrator
	configPaths := []string{
		"examples/03-symbolic-planner/config.yaml",
		"testdata/config_valid.yaml",
	}

	var configData []byte
	for _, path := range configPaths {
		data, err := ioutil.ReadFile(path)
		if err == nil {
			configData = data
			fmt.Printf("Loaded configuration from %s\n", path)
			break
		}
	}

	if configData == nil {
		fmt.Println("Note: No planner config file found")
		fmt.Println("To use this example, create a config.yaml with a planner-based orchestrator")
		fmt.Println()
		fmt.Println("Example config snippet:")
		fmt.Println("  orchestrators:")
		fmt.Println("    - name: symbolic-planner")
		fmt.Println("      provider: symbolic-planner")
		fmt.Println("      options:")
		fmt.Println("        reasoner: my_reasoner")
		fmt.Println("        tools:")
		fmt.Println("          - retriever")
		fmt.Println("          - llm")
		return
	}

	fmt.Println()

	// Get the global registry
	reg := registry.Global()

	// Load and build from config
	fmt.Println("Loading and building planner orchestrator...")
	fmt.Println()

	orch, err := sdk.LoadWithRegistry(ctx, configData, reg)
	if err != nil {
		fmt.Printf("Note: Failed to load configuration: %v\n", err)
		fmt.Println("This is OK for a demo - ensure GOOGLE_API_KEY is set if using Google LLM")
		return
	}

	defer func() {
		if err := orch.Close(ctx); err != nil {
			log.Printf("Warning: Error closing orchestrator: %v", err)
		}
	}()

	// Execute a query
	fmt.Println("✓ Planner orchestrator built successfully!")
	fmt.Println()

	query := core.Query{
		Text: "What are the latest advances in machine learning?",
		Meta: map[string]any{
			"user_id": "demo-user",
			"session": "example-session",
		},
	}

	fmt.Printf("Executing query: %s\n", query.Text)
	fmt.Println()

	answer, err := orch.Execute(ctx, "session-123", query)
	if err != nil {
		log.Printf("Note: Query execution failed (expected if not fully configured): %v\n", err)
		fmt.Println()
		fmt.Println("In a real application, the planner would:")
		fmt.Println("  1. Generate a multi-step execution plan")
		fmt.Println("  2. Execute each step using the specified tools")
		fmt.Println("  3. Combine results into a final answer")
		return
	}

	fmt.Printf("\nAnswer: %s\n", answer.Text)
	if len(answer.Citations) > 0 {
		fmt.Println("\nCitations:")
		for _, citation := range answer.Citations {
			fmt.Printf("  - %s (Source: %s)\n", citation.ID, citation.Source)
		}
	}
}
