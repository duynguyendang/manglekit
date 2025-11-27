package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/duynguyendang/manglekit/v1/core"
	"github.com/duynguyendang/manglekit/v1/internal/registry"
	_ "github.com/duynguyendang/manglekit/v1/providers/all" // Auto-registers all standard providers
	"github.com/duynguyendang/manglekit/sdk"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Note: .env file not found, will fall back to environment variables")
	}

	ctx := context.Background()

	// Load from YAML config file (config-first approach - recommended)
	// This demonstrates the best-practice declarative setup
	configPath := "examples/01-programmatic-setup/config.yaml"
	configData, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Printf("Note: config.yaml not found at %s\n", configPath)
		log.Println("For config-first setup, create config.yaml in examples/01-programmatic-setup/")
		return
	}

	log.Printf("Loading configuration from %s\n", configPath)

	// Get the global registry with all providers registered
	reg := registry.Global()

	// Use sdk.LoadWithRegistry to load YAML config
	// This automatically uses all registered providers from providers/all import
	orch, err := sdk.LoadWithRegistry(ctx, configData, reg)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Ensure proper cleanup
	defer func() {
		if err := orch.Close(ctx); err != nil {
			log.Printf("Warning: Error closing orchestrator: %v", err)
		}
	}()

	// Execute a query
	query := core.Query{
		Text: "What is MangleKit?",
	}
	log.Printf("\nExecuting query: %s\n", query.Text)

	answer, err := orch.Execute(ctx, "session-123", query)
	if err != nil {
		log.Fatalf("Failed to execute query: %v", err)
	}

	fmt.Printf("\nAnswer: %s\n", answer.Text)
	if len(answer.Citations) > 0 {
		fmt.Println("\nCitations:")
		for _, citation := range answer.Citations {
			fmt.Printf("  - %s (Source: %s)\n", citation.ID, citation.Source)
		}
	}
}
