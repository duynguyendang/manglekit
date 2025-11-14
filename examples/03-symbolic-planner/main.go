// Package main demonstrates how to use the symbolic planner in Manglekit.
// The symbolic planner uses a reasoner to generate multi-step execution plans.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
	obslogger "github.com/duynguyendang/manglekit/internal/logger"
	"github.com/duynguyendang/manglekit/internal/providers/planners/symbolic"
)

// This example demonstrates using the symbolic planner directly without YAML configuration.

// mockReasoner is a simple reasoner that returns a predefined plan.
type mockReasoner struct{}

func (m *mockReasoner) Execute(ctx context.Context, req core.ReasonerRequest) (core.ReasonerResponse, error) {
	// In a real implementation, this would perform symbolic reasoning
	// based on the input facts and return a logical plan.
	// For this demo, we return a simple two-step plan.
	
	output := map[string]any{
		// Step 0: Search for documents
		"plan_tool_0":   "retriever",
		"plan_params_0": `{"query": "machine learning", "topK": 5}`,
		"plan_reason_0": "Search for relevant documents about the query topic",
		
		// Step 1: Generate summary
		"plan_tool_1":   "llm",
		"plan_params_1": `{"prompt": "Summarize the following documents", "maxTokens": 500}`,
		"plan_reason_1": "Generate a comprehensive summary from retrieved documents",
	}
	
	return core.ReasonerResponse{Output: output}, nil
}

func main() {
	ctx := context.Background()
	
	// Create a mock reasoner
	reasoner := &mockReasoner{}
	
	// Create dependencies for the planner
	logger := obslogger.NewStdLogger()
	deps := diapi.PlannerDeps{
		CoreDeps: diapi.CoreDeps{
			Obs: core.Observability{
				Logger: logger,
			},
		},
		Reasoners: map[string]core.Reasoner{
			"demo-reasoner": reasoner,
		},
	}
	
	// Create planner options
	opts := &symbolic.Options{
		ReasonerName: "demo-reasoner",
	}
	
	// Create the planner
	planner, err := symbolic.NewFactory(deps, opts)
	if err != nil {
		log.Fatalf("Failed to create planner: %v", err)
	}
	
	// Create a query
	query := core.Query{
		Text: "What are the latest advances in machine learning?",
		Meta: map[string]any{
			"user_id": "demo-user",
			"session": "example-session",
		},
	}
	
	// Generate a plan
	fmt.Println("Generating plan for query:", query.Text)
	fmt.Println()
	
	plan, err := planner.Plan(ctx, query)
	if err != nil {
		log.Fatalf("Failed to generate plan: %v", err)
	}
	
	// Display the plan
	fmt.Printf("Generated plan with %d steps:\n\n", len(plan.Steps))
	
	for i, step := range plan.Steps {
		fmt.Printf("Step %d:\n", i+1)
		fmt.Printf("  Tool:   %s\n", step.Tool)
		fmt.Printf("  Reason: %s\n", step.Reason)
		
		if len(step.Params) > 0 {
			paramsJSON, _ := json.MarshalIndent(step.Params, "  ", "  ")
			fmt.Printf("  Params: %s\n", string(paramsJSON))
		}
		fmt.Println()
	}
	
	// In a real application, you would execute these steps using the orchestrator
	fmt.Println("✓ Plan generated successfully!")
	fmt.Println("\nIn a real application, this plan would be executed by the orchestrator,")
	fmt.Println("which would call each tool in sequence with the specified parameters.")
	
	os.Exit(0)
}
