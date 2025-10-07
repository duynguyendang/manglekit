package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	_ "github.com/duynguyendang/manglekit/internal/providers/schemaparsers/rdf" // register the new rdf parser
	_ "github.com/duynguyendang/manglekit/providers/all"                     // register all standard providers
)

func main() {
	ctx := context.Background()

	builder, err := manglekit.NewBuilderFromYAML("./config.yaml")
	if err != nil {
		log.Fatalf("Failed to create builder from yaml: %v", err)
	}

	orch, err := builder.Build()
	if err != nil {
		log.Fatalf("Failed to build orchestrator: %v", err)
	}

	// === Test Case 1: Alice requests access to doc123 (should be allowed) ===
	fmt.Println("--- Running Test Case 1: Alice requests doc123 ---")
	aliceQuery := core.Query{
		Text: "Can I access this document?",
		Meta: map[string]any{
			"user_context": map[string]any{
				"name":              "<http://example.org/ontology#alice>",
				"document":          "<http://example.org/ontology#doc123>",
				"permission_pred":   "<http://example.org/ontology#hasPermission>",
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
				"name":              "<http://example.org/ontology#bob>",
				"document":          "<http://example.org/ontology#doc123>",
				"permission_pred":   "<http://example.org/ontology#hasPermission>",
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