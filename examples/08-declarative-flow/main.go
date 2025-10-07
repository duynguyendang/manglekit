package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	_ "github.com/duynguyendang/manglekit/providers/all"
)

func main() {
	if os.Getenv("OPENAI_API_KEY") == "" {
		log.Fatal("OPENAI_API_KEY environment variable not set")
	}

	ctx := context.Background()

	// The builder reads the config.yaml file, which specifies the "declarative"
	// orchestrator, defines all the tools, and points to the flow.dlog file.
	builder, err := manglekit.NewBuilderFromYAML("./config.yaml")
	if err != nil {
		log.Fatalf("Failed to create builder from YAML: %v", err)
	}

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