// Package main for 06-schema-validation demonstrates how Mangle's rule engine
// can be used for powerful input and output schema validation.
//
// This example defines a custom `FactConverter` to turn an incoming request body
// into facts. It then uses a Mangle rule (`rules/policy.dlog`) to check these
// facts against a schema definition loaded from a JSON Schema file
// (`user.schema.json`). This showcases how Mangle can enforce data integrity,
// security, and business logic by validating data structures at runtime.
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
	_ "github.com/duynguyendang/manglekit/providers/all" // Register all standard providers
	"github.com/google/mangle/ast"
	"github.com/joho/godotenv"
)

// requestConverter turns a map from a core.Query into `request_field/2` facts.
type requestConverter struct{}

// This init function registers our custom converter with the Manglekit registry.
func init() {
	manglekit.Register("requestConverter", func(ctx context.Context, options any, deps manglekit.FactoryDeps) (any, error) {
		return &requestConverter{}, nil
	})
}

func (c *requestConverter) Predicates() []ast.PredicateSym {
	return []ast.PredicateSym{{Symbol: "request_field", Arity: 2}}
}

// ToFacts converts the `Meta["request_body"]` of a query into facts.
// This simplified version only handles string values.
func (c *requestConverter) ToFacts(input any) ([]ast.Atom, error) {
	query, ok := input.(core.Query)
	if !ok {
		return nil, fmt.Errorf("requestConverter: expected core.Query, got %T", input)
	}

	requestBody, ok := query.Meta["request_body"].(map[string]any)
	if !ok {
		return nil, nil // No data, no facts.
	}

	var facts []ast.Atom
	for key, value := range requestBody {
		if v, ok := value.(string); ok {
			facts = append(facts, ast.NewAtom("request_field", ast.String(key), ast.String(v)))
		}
	}
	return facts, nil
}

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
		panic(fmt.Errorf("failed to create builder from yaml: %w", err))
	}

	orch, err := builder.Build(ctx)
	if err != nil {
		panic(fmt.Errorf("failed to build orchestrator: %w", err))
	}
	defer orch.Close(ctx)

	// 2. Define test cases for the simplified string-based rule.
	testCases := []struct {
		name        string
		requestBody map[string]any
		expectDeny  bool
	}{
		{name: "Valid user", requestBody: map[string]any{"email": "test@example.com", "status": "active"}, expectDeny: false},
		{name: "Blocked user", requestBody: map[string]any{"email": "bad@example.com", "status": "blocked"}, expectDeny: true},
	}

	fmt.Println("--- Running Schema Validation Tests ---")
	allPassed := true

	// 3. Run the tests.
	for _, tc := range testCases {
		query := core.Query{
			Text: "A test query",
			Meta: map[string]any{"request_body": tc.requestBody},
		}

		result, err := orch.Execute(ctx, "session-1", query)
		wasDenied := errors.Is(err, core.ErrDenied)

		if wasDenied == tc.expectDeny {
			fmt.Printf("Test Case: %q - PASSED\n", tc.name)
			if wasDenied {
				deniedReasons, _ := result.Meta["mangle_denied_reasons"]
				fmt.Printf("  > Denied as expected. Reasons: %v\n", deniedReasons)
			}
		} else {
			allPassed = false
			fmt.Printf("Test Case: %q - FAILED\n", tc.name)
			fmt.Printf("  > Expected deny: %v, but got: %v. Error: %v\n", tc.expectDeny, wasDenied, err)
		}
	}

	if !allPassed {
		fmt.Println("\nSome tests failed.")
	} else {
		fmt.Println("\nAll tests passed.")
	}
}
