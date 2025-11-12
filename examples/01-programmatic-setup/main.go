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
	err := godotenv.Load()
	if err != nil {
		log.Println("Note: .env file not found, will fall back to environment variables")
	}

	ctx := context.Background()

	// 1. Create a new programmatic builder
	builder, err := sdk.NewBuilder(ctx)
	if err != nil {
		log.Fatalf("Failed to create builder: %v", err)
	}

	// 2. Add components programmatically
	// BM25 Retriever
	bm25Opts := &bm25.BM25Options{
		Path: "testdata/acme-corp",
	}

	// OpenAI LLM
	openAIOpts := &llm.OpenAIOptions{
		Model:          "gpt-3.5-turbo",
		APIKey:         os.Getenv("OPENAI_API_KEY"),
		SkipModelCheck: true,
	}

	// Mangle RuleSet
	mangleOpts := &core.MangleOptions{
		Path: []string{"examples/rules/acme-rules.dlog"},
	}

	// Sandwich Orchestrator
	sandwichOpts := &sandwich.Options{
		LLM:       "openai",
		Retriever: "bm25",
		RuleSet:   "mangle",
	}

	builder.WithOptions("bm25", bm25Opts).
		WithOptions("openai", openAIOpts).
		WithOptions("mangle", mangleOpts).
		WithOptions("sandwich", sandwichOpts)

	// 3. Build the orchestrator
	orch, _, err := builder.Build(ctx, "sandwich", "")
	if err != nil {
		log.Fatalf("Failed to build orchestrator: %v", err)
	}
	defer orch.Close(ctx)

	// 4. Execute a query
	query := core.Query{Text: "manglekit"}
	answer, err := orch.Execute(ctx, "session-123", query)
	if err != nil {
		log.Fatalf("Failed to execute query: %v", err)
	}

	fmt.Printf("Answer: %s\n", answer.Text)
}
