package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"runtime"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	_ "github.com/duynguyendang/manglekit/providers/all"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load() // Load .env file if present.
	ctx := context.Background()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatalf("failed to get current file path")
	}
	dir := filepath.Dir(currentFile)
	configPath := filepath.Join(dir, "config.yaml")

	// 1. Load the base configuration from YAML.
	// This will set up the Mangle rules and the LLM.
	builder, err := manglekit.NewBuilderFromYAML(configPath)
	if err != nil {
		log.Fatalf("Failed to create builder from yaml: %v", err)
	}
	log.Println("Successfully created builder from YAML.")

	// 2. Build the orchestrator.

	// The Build() method now reads the orchestrator type from the config.
	// It sees "declarative", builds the dependency graph of tools, and then
	// instantiates the DeclarativeOrchestrator with the built tools and ruleset.
	orchestrator, err := builder.Build()
	if err != nil {
		log.Fatalf("Failed to build orchestrator: %v", err)
	}

	log.Println("Successfully built declarative orchestrator.")

	// Run a query through the pipeline defined in flow.dlog.
	query := core.Query{Text: "What is the capital of France?"}
	log.Printf("Executing query: %q\n", query.Text)

	answer, err := orchestrator.Run(ctx, query)
	if err != nil {
		log.Fatalf("Orchestrator run failed: %v", err)
	}

	fmt.Println("--------------------")
	fmt.Printf("Final Answer: %s\n", answer.Text)
	fmt.Println("--------------------")

	// Demonstrate the conditional skip logic defined in flow.dlog.
	querySkip := core.Query{Text: "What is the capital of France? (no_llm)"}
	log.Printf("\nExecuting query that should skip the LLM stage: %q\n", querySkip.Text)

	answerSkip, err := orchestrator.Run(ctx, querySkip)
	if err != nil {
		log.Fatalf("Orchestrator run failed: %v", err)
	}

	fmt.Println("--------------------")
	fmt.Printf("Final Answer (should be empty): '%s'\n", answerSkip.Text)
	fmt.Println("--------------------")
}
