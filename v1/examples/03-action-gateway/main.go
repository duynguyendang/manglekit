package main

import (
	"context"
	"fmt"
	"os"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
	"github.com/duynguyendang/manglekit/internal/logger"
	"github.com/duynguyendang/manglekit/internal/providers/rules/mangle"
	"github.com/duynguyendang/manglekit/pipeline/sandwich"
)

// A. The State (Data Model)
type MathRequest struct {
	A  int    `json:"a" mangle:"arg_a"`
	B  int    `json:"b" mangle:"arg_b"`
	Op string `json:"op" mangle:"op"`
}

// B. The Action (Business Logic)
type CalculatorAction struct{}

func (c *CalculatorAction) Execute(ctx context.Context, input any) (any, error) {
	// Input is expected to be a core.Query, but the logic data is in Meta["math_req"]
	q, ok := input.(core.Query)
	if !ok {
		return nil, fmt.Errorf("expected core.Query input, got %T", input)
	}

	reqData, ok := q.Meta["math_req"]
	if !ok {
		return nil, fmt.Errorf("missing 'math_req' in query metadata")
	}

	req, ok := reqData.(MathRequest)
	if !ok {
		return nil, fmt.Errorf("expected MathRequest in metadata, got %T", reqData)
	}

	// Important: Do not check for "Division by Zero" inside the Go code.
	// The Logic Engine (Mangle) should catch this before we even get here.
	var result int
	switch req.Op {
	case "add":
		result = req.A + req.B
	case "sub":
		result = req.A - req.B
	case "mul":
		result = req.A * req.B
	case "div":
		// This would panic if B is 0, but policy protects us.
		result = req.A / req.B
	default:
		return nil, fmt.Errorf("unknown operation: %s", req.Op)
	}

	return fmt.Sprintf("Result: %d", result), nil
}

// C. The Policy (Datalog Rules)
// We add dummy facts to implicitly declare the EDB predicates since we are using FileFirst=true
// and 'Decl' syntax is not supported in the parser.
const policyRules = `
op("init", "noop").
arg_b("init", -1).

deny("division by zero", "policy") :- op(E, "div"), arg_b(E, 0).

deny("unknown operation", "policy") :- op(E, Op), Op != "add", Op != "sub", Op != "mul", Op != "div", Op != "noop".
`

func main() {
	ctx := context.Background()
	obs := core.Observability{
		Logger: logger.NewStdLogger(),
	}

	// Initialize the CalculatorAction
	calcAction := &CalculatorAction{}

	// Initialize Mangle RuleSet with the policy
	// We need to write the policy to a temp file because Mangle provider reads from files
	tmpFile, err := os.CreateTemp("", "policy_*.dlog")
	if err != nil {
		panic(err)
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.WriteString(policyRules); err != nil {
		panic(err)
	}
	if err := tmpFile.Close(); err != nil {
		panic(err)
	}

	ruleset, err := mangle.New(ctx, core.MangleOptions{
		Path:      []string{tmpFile.Name()},
		FileFirst: true, // Enable file-first mode
	}, nil)
	if err != nil {
		panic(fmt.Errorf("failed to create ruleset: %w", err))
	}

	// Wiring & Execution: Initialize Sandwich orchestrator manually
	deps := diapi.SandwichDeps{
		CoreDeps: diapi.CoreDeps{Obs: obs},
		Action:   calcAction,
		RuleSet:  ruleset,
		// No LLM, Reranker, or StateProvider needed for this example
	}

	// Create factory and build orchestrator
	factory := sandwich.NewFactory()
	orchAny, err := factory.Build(ctx, deps, &sandwich.Options{})
	if err != nil {
		panic(fmt.Errorf("failed to build orchestrator: %w", err))
	}
	orch := orchAny.(core.Orchestrator)
	defer orch.Close(ctx)

	fmt.Println("--- Manglekit Action Gateway Example ---")
	fmt.Println("Demonstrating Universal Governance: Using Manglekit as a Logic Firewall.")
	fmt.Println()

	// Run Test Case 1 (Valid): 10 / 2
	fmt.Println("Test Case 1: 10 / 2 (Valid)")
	req1 := MathRequest{A: 10, B: 2, Op: "div"}
	q1 := core.Query{
		Text: "Execute 10 / 2",
		Meta: map[string]any{"math_req": req1},
	}

	ans1, err := orch.Execute(ctx, "session-1", q1)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		// The result is in Meta["action_result"] because it's not a RetrieveResult
		if res, ok := ans1.Meta["action_result"]; ok {
			fmt.Printf("Success: %v\n", res)
		} else {
			fmt.Printf("Success (No result returned?): %+v\n", ans1)
		}
	}
	fmt.Println()

	// Run Test Case 2 (Invalid): 10 / 0
	fmt.Println("Test Case 2: 10 / 0 (Should be blocked by Policy)")
	req2 := MathRequest{A: 10, B: 0, Op: "div"}
	q2 := core.Query{
		Text: "Execute 10 / 0",
		Meta: map[string]any{"math_req": req2},
	}

	_, err = orch.Execute(ctx, "session-2", q2)
	if err != nil {
		fmt.Printf("Caught Expected Error: %v\n", err)
	} else {
		fmt.Println("Error: Operation was NOT blocked!")
	}
}
