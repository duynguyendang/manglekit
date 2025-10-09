package main

import (
	"context"
	"errors"
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
	defer orchestrator.Close(ctx)

	log.Println("Successfully built declarative orchestrator.")

	runQuery := func(label string, q core.Query) {
		log.Printf("Executing %s: %q\n", label, q.Text)
		answer, err := orchestrator.Run(ctx, q)

		switch {
		case errors.Is(err, core.ErrDenied):
			fmt.Println("--------------------")
			fmt.Printf("%s denied: %v\n", label, answer.Meta["denial_reason"])
			if ruleResults, ok := answer.Meta["rule_results"]; ok {
				fmt.Printf("Rule results: %#v\n", ruleResults)
			}
			fmt.Println("--------------------")
			return
		case err != nil:
			log.Fatalf("Orchestrator run failed: %v", err)
		}

		fmt.Println("--------------------")
		fmt.Printf("%s Answer: %s\n", label, answer.Text)
		if len(answer.Citations) > 0 {
			fmt.Println("Citations:")
			for _, c := range answer.Citations {
				fmt.Printf(" - %s (%s)\n", c.ID, c.Source)
			}
		}
		if ruleResults, ok := answer.Meta["rule_results"]; ok {
			fmt.Printf("Rule results: %#v\n", ruleResults)
		}
		if redactions, ok := answer.Meta["redactions"]; ok {
			fmt.Printf("Redactions applied: %#v\n", redactions)
		}
		fmt.Println("--------------------")
	}

	runQuery("Employee", core.Query{
		Text: "What is the capital of France?",
		Meta: map[string]any{
			"user_context": map[string]any{"role": "employee"},
		},
	})

	runQuery("Guest", core.Query{
		Text: "What is the capital of France?",
		Meta: map[string]any{
			"user_context": map[string]any{"role": "guest"},
		},
	})

	runQuery("No LLM", core.Query{
		Text: "What is the capital of France? (no_llm)",
		Meta: map[string]any{
			"user_context": map[string]any{"role": "guest"},
		},
	})
}
