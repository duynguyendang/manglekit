package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	// Blank import to ensure all standard providers, including Mangle, are registered.
	// Our custom converter is registered via register.go's init() function.
	_ "github.com/duynguyendang/manglekit/providers/all"
)

func main() {
	// Use the builder to load the pipeline from the YAML configuration.
	// The builder will read the config and correctly wire up our custom converter.
	builder, err := manglekit.NewBuilderFromYAML("examples/04-chat-w-data/config.yaml")
	if err != nil {
		log.Fatalf("Failed to create builder from YAML: %v", err)
	}

	pipeline, err := builder.Build()
	if err != nil {
		log.Fatalf("Failed to build pipeline: %v", err)
	}

	// Define test cases. The metadata now only contains the dynamic 'request' info.
	// User attributes are now resolved from the static facts in facts.dlog.
	testCases := []struct {
		description string
		query       core.Query
	}{
		{
			description: "Sales analyst 'alice' asking for a sales document (should be allowed).",
			query: core.Query{
				Meta: map[string]any{
					"request": map[string]string{"user": "alice", "doc_id": "A123"},
				},
			},
		},
		{
			description: "Sales analyst 'alice' asking for a marketing document (should be denied).",
			query: core.Query{
				Meta: map[string]any{
					"request": map[string]string{"user": "alice", "doc_id": "B456"},
				},
			},
		},
		{
			description: "User 'bob' asking for a document with 'normal' confidentiality (should be allowed).",
			query: core.Query{
				Meta: map[string]any{
					"request": map[string]string{"user": "bob", "doc_id": "A123"},
				},
			},
		},
		{
			description: "User 'alice' with an explicit doc_id grant asking for that doc (should be allowed).",
			query: core.Query{
				Meta: map[string]any{
					// This works because one of the static facts is:
					// user_attribute("alice", "doc_id", "A123").
					// And the request is for "A123".
					"request": map[string]string{"user": "alice", "doc_id": "A123"},
				},
			},
		},
		{
			description: "User 'dave' with no grants asking for a restricted document (should be denied).",
			query: core.Query{
				Meta: map[string]any{
					"request": map[string]string{"user": "dave", "doc_id": "S777"},
				},
			},
		},
	}

	fmt.Println("--- Running Access Control Scenarios ---")

	for _, tc := range testCases {
		fmt.Printf("\n--- \nDescription: %s\n", tc.description)
		_, err := pipeline.Run(context.Background(), tc.query)

		if err != nil {
			if errors.Is(err, core.ErrDenied) {
				fmt.Printf("Result: Access DENIED. Reason: %v\n", err)
			} else {
				log.Printf("Unexpected error executing pipeline: %v", err)
			}
		} else {
			fmt.Println("Result: Access ALLOWED.")
		}
	}
	fmt.Println("\n--- All scenarios complete. ---")
}