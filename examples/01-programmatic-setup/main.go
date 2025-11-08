package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/providers/llm"
	"github.com/duynguyendang/manglekit/internal/providers/retrievers/bm25"
	"github.com/duynguyendang/manglekit/pipeline/sandwich"
	"github.com/duynguyendang/manglekit/sdk"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from a .env file.
	err := godotenv.Load()
	if err != nil {
		log.Printf("Warning: could not load .env file: %v", err)
	}

	ctx := context.Background()

	// Create a new programmatic builder.
	builder, err := sdk.NewBuilder(ctx)
	if err != nil {
		log.Fatalf("Failed to create builder: %v", err)
	}

	// 1. Configure a bm25 retriever.
	bm25Opts := &bm25.BM25Options{
		Path: "examples/01-programmatic-setup/docs",
		TopK: 3,
	}

	// 2. Configure a Google LLM.
	googleOpts := &llm.GoogleOptions{
		Model: "gemini-1.5-flash",
	}

	// 3. Configure a Mangle ruleset.
	// Create a dummy rules file.
	dummyRuleFile, err := os.Create("examples/01-programmatic-setup/rules.dlog")
	if err != nil {
		log.Fatalf("Failed to create dummy rule file: %v", err)
	}
	dummyRuleFile.Close()

	mangleOpts := &core.MangleOptions{
		Path: []string{"examples/01-programmatic-setup/rules.dlog"},
	}

	// 4. Configure the sandwich orchestrator.
	sandwichOpts := &sandwich.Options{
		LLM:       "google",
		Retriever: "bm25",
		RuleSet:   "mangle",
	}

	// Register components with the builder.
	err = builder.With(
		bm25Opts,
		googleOpts,
		mangleOpts,
		sandwichOpts,
	)
	if err != nil {
		log.Fatalf("Failed to add components to builder: %v", err)
	}

	// Build the orchestrator.
	orch, _, err := builder.Build(ctx, "sandwich", "", "")
	if err != nil {
		log.Fatalf("Failed to build orchestrator: %v", err)
	}
	defer orch.Close(ctx)

	// Execute a sample query.
	query := core.Query{Text: "What is MangleKit?"}
	answer, err := orch.Execute(ctx, "session-123", query)
	if err != nil {
		log.Fatalf("Failed to execute query: %v", err)
	}

	fmt.Printf("Answer: %s\n", answer.Text)
}
