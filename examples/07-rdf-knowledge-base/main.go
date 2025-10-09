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
	_ "github.com/duynguyendang/manglekit/providers/all" // register all standard providers
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load() // load .env file if exists
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

	orch, err := builder.Build()
	if err != nil {
		log.Fatalf("Failed to build orchestrator: %v", err)
	}
	defer orch.Close(ctx)

	// === Test Case 1: Alice requests access to doc123 (should be allowed) ===
	fmt.Println("--- Running Test Case 1: Alice requests doc123 ---")
	aliceQuery := core.Query{
		Text: "Can I access this document?",
		Meta: map[string]any{
			"user_context": map[string]any{
				"name":            "<http://example.org/ontology#alice>",
				"document":        "<http://example.org/ontology#doc123>",
				"permission_pred": "<http://example.org/ontology#hasPermission>",
			},
		},
	}
	runTest("Alice (Allowed)", ctx, orch, aliceQuery, false)

	// === Test Case 2: Bob requests access to doc123 (should be denied) ===
	fmt.Println("\n--- Running Test Case 2: Bob requests doc123 ---")
	bobQuery := core.Query{
		Text: "Can I access this document?",
		Meta: map[string]any{
			"user_context": map[string]any{
				"name":            "<http://example.org/ontology#bob>",
				"document":        "<http://example.org/ontology#doc123>",
				"permission_pred": "<http://example.org/ontology#hasPermission>",
			},
		},
	}
	runTest("Bob (Denied)", ctx, orch, bobQuery, true)
}

func runTest(name string, ctx context.Context, orch core.Orchestrator, query core.Query, expectDeny bool) {
	result, err := orch.Run(ctx, query)
	wasDenied := errors.Is(err, core.ErrDenied)

	if wasDenied == expectDeny {
		fmt.Printf("Test Case: %q - PASSED\n", name)
		if wasDenied {
			deniedReasons, _ := result.Meta["mangle_denied_reasons"]
			fmt.Printf("  > Decision: Denied. Justification: %v\n", deniedReasons)
		} else {
			fmt.Println("  > Decision: Allowed.")
		}
	} else {
		fmt.Printf("Test Case: %q - FAILED\n", name)
		fmt.Printf("  > Expected deny: %v, but got: %v. Error: %v\n", expectDeny, wasDenied, err)
	}
}
