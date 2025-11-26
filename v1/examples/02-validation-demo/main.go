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

	// DEMO: Show configuration validation
	// This example demonstrates what happens when you try to build
	// a pipeline with missing dependencies

	ctx := context.Background()

	fmt.Println("=== Configuration Validation Demo ===")
	fmt.Println()

	// Load a config file
	configPath := "testdata/config_valid.yaml"
	configData, err := ioutil.ReadFile(configPath)
	if err != nil {
		// Try an alternative path
		configPath = "config.yaml"
		configData, err = ioutil.ReadFile(configPath)
		if err != nil {
			fmt.Printf("Note: No config file found at %s\n", configPath)
			fmt.Println("This demo shows what happens when configs are loaded and validated")
			fmt.Println("For a real run, provide a valid YAML config file")
			return
		}
	}

	fmt.Printf("Loading configuration from %s\n", configPath)
	fmt.Println()

	// Get the global registry
	reg := registry.Global()

	// Try to build from config
	fmt.Println("Attempting to load and validate configuration...")
	fmt.Println()

	orch, err := sdk.LoadWithRegistry(ctx, configData, reg)
	if err != nil {
		fmt.Printf("❌ Configuration failed to load:\n    %v\n\n", err)
		fmt.Println("This error shows that missing dependencies or invalid")
		fmt.Println("configurations are caught during validation.")
		fmt.Println()

		// Show what might cause validation errors
		fmt.Println("Common validation errors:")
		fmt.Println("  1. Missing GOOGLE_API_KEY environment variable (required for Google LLM)")
		fmt.Println("  2. Reference to non-existent component in orchestrator config")
		fmt.Println("  3. Invalid options for a provider")
		fmt.Println()
		return
	}

	fmt.Println("✓ Configuration loaded and validated successfully!")
	fmt.Println()

	// If we got here, everything is configured correctly
	defer func() {
		if err := orch.Close(ctx); err != nil {
			log.Printf("Warning: Error closing orchestrator: %v", err)
		}
	}()

	// Try a simple query
	query := core.Query{
		Text: "What is MangleKit?",
	}
	log.Printf("Executing query: %s\n", query.Text)

	answer, err := orch.Execute(ctx, "session-123", query)
	if err != nil {
		log.Printf("Note: Query execution failed (this is OK for demo): %v\n", err)
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
