package main

import (
	"context"
	"fmt"
	"log"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
	"github.com/duynguyendang/manglekit/internal/logger"
	"github.com/duynguyendang/manglekit/internal/providers/rules/mangle"
	"github.com/duynguyendang/manglekit/internal/testproviders/mock"
	"github.com/duynguyendang/manglekit/pipeline/sandwich"
)

// RequestCtx defines the structure for our request metadata, with mangle tags
// for fact projection.
type RequestCtx struct {
	TokenCount int  `mangle:"token_count"`
	IsSecret   bool `mangle:"is_secret"`
}

func main() {
	ctx := context.Background()

	// 1. Create mock actions (using mock retrievers adapted to core.Action)
	geminiAction := &core.RetrieverAction{
		Retriever: mock.NewRetriever(map[string]string{"default": "[Gemini Flash] Processing query..."}),
	}
	localAction := &core.RetrieverAction{
		Retriever: mock.NewRetriever(map[string]string{"secret": "[Local Llama] Processing secret query..."}),
	}
	gpt4Action := &core.RetrieverAction{
		Retriever: mock.NewRetriever(map[string]string{"complex": "[GPT-4] Processing complex query..."}),
	}

	// 2. Create the RuleSet programmatically
	ruleset, err := mangle.New(ctx, core.MangleOptions{
		Path:      []string{"examples/05-smart-router/policy.dlog"},
		FileFirst: true, // Allow .dlog files to define their own predicates
	}, nil)
	if err != nil {
		log.Fatalf("failed to create ruleset: %v", err)
	}

	// 3. Assemble dependencies for the Sandwich orchestrator
	deps := diapi.SandwichDeps{
		CoreDeps: diapi.CoreDeps{
			Obs: core.Observability{Logger: logger.NewStdLogger()},
		},
		Action:  geminiAction, // Default action
		RuleSet: ruleset,
		SubActions: map[string]core.Action{
			"local": localAction,
			"gpt4":  gpt4Action,
		},
		// Other dependencies are nil as they are not needed for this example
		LLM:      nil,
		Reranker: nil,
	}

	// 4. Build the orchestrator using its factory
	factory := sandwich.NewFactory()
	built, err := factory.Build(ctx, deps, &sandwich.Options{})
	if err != nil {
		log.Fatalf("failed to build orchestrator: %v", err)
	}
	orchestrator := built.(core.Orchestrator)
	defer orchestrator.Close(ctx)

	// 5. Run test cases
	testCases := []struct {
		name              string
		query             core.Query
		expectedAction    string
		expectedDoc       string
		expectedQueryText string
	}{
		{
			name: "Default case (Gemini Flash)",
			query: core.Query{
				Text: "What is the capital of France?",
				Meta: map[string]any{"request_context": &RequestCtx{TokenCount: 100, IsSecret: false}},
			},
			expectedAction:    "default",
			expectedDoc:       "[Gemini Flash] Processing query...",
			expectedQueryText: "default", // The mock retriever needs a matching key
		},
		{
			name: "Secret case (Local Llama)",
			query: core.Query{
				Text: "Tell me the secret formula.",
				Meta: map[string]any{"request_context": &RequestCtx{TokenCount: 200, IsSecret: true}},
			},
			expectedAction:    "local",
			expectedDoc:       "[Local Llama] Processing secret query...",
			expectedQueryText: "secret",
		},
		{
			name: "Complex case (GPT-4)",
			query: core.Query{
				Text: "Explain quantum mechanics in detail.",
				Meta: map[string]any{"request_context": &RequestCtx{TokenCount: 2000, IsSecret: false}},
			},
			expectedAction:    "gpt4",
			expectedDoc:       "[GPT-4] Processing complex query...",
			expectedQueryText: "complex",
		},
	}

	for _, tc := range testCases {
		fmt.Printf("--- Running test case: %s ---\n", tc.name)
		// The mock retrievers are keyed on specific strings, so we override the query text for the test
		queryToRun := tc.query
		queryToRun.Text = tc.expectedQueryText

		answer, err := orchestrator.Execute(ctx, "session-123", queryToRun)
		if err != nil {
			log.Printf("  ERROR: %v", err)
			fmt.Println("  FAIL")
		} else {
			var pass = true
			fmt.Printf("  Executed Action: '%s'\n", answer.Meta["executed_action"])
			if answer.Meta["executed_action"] != tc.expectedAction {
				fmt.Printf("  FAIL: Expected action '%s'\n", tc.expectedAction)
				pass = false
			}

			if result, ok := answer.Meta["action_result"].(core.RetrieveResult); ok {
				if len(result.Docs) > 0 {
					fmt.Printf("  Result: %s\n", result.Docs[0].Text)
					if result.Docs[0].Text != tc.expectedDoc {
						fmt.Printf("  FAIL: Expected doc '%s'\n", tc.expectedDoc)
						pass = false
					}
				} else {
					fmt.Println("  FAIL: No documents returned")
					pass = false
				}
			} else {
				fmt.Printf("  FAIL: Unexpected result type: %T\n", answer.Meta["action_result"])
				pass = false
			}

			if pass {
				fmt.Println("  PASS")
			}
		}
		fmt.Println()
	}
}
