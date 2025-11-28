package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/engine"
	"github.com/duynguyendang/manglekit/guard"
)

// EchoAction is a simple action that returns the input as output.
type EchoAction struct{}

func (a *EchoAction) Execute(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	// Simple echo: payload remains same, new envelope created
	output := core.NewEnvelope(input.Payload)
	return output, nil
}

func (a *EchoAction) Metadata() core.ActionMetadata {
	return core.ActionMetadata{
		Name: "Echo",
		Type: "Utility",
	}
}

func main() {
	// 1. Setup Engine
	eng := engine.New()

	// 2. Define Policy: Block output if it has "secret" label
	// We use has_label("init", "init"). to implicitly declare the predicate.
	policyContent := `
	has_label("init", "init").
	deny("Output") :- has_label("Output", "secret").
	`
	err := os.WriteFile("taint_policy.dlog", []byte(policyContent), 0644)
	if err != nil {
		fmt.Printf("Failed to write policy file: %v\n", err)
		os.Exit(1)
	}
	defer os.Remove("taint_policy.dlog")

	if err := eng.LoadFromPath("taint_policy.dlog"); err != nil {
		fmt.Printf("Failed to load policy: %v\n", err)
		os.Exit(1)
	}

	// 3. Create Guarded Action
	echo := &EchoAction{}
	guardedEcho := guard.New(echo, eng, "closed")

	// 4. Test Case 1: Input has "secret" label. Output should inherit it and be blocked.
	fmt.Println("Test Case 1: Input with 'secret' label")
	input1 := core.NewEnvelope("my secret data")
	input1.AddLabel("secret")

	result1, err := guardedEcho.Execute(context.Background(), input1)
	if err == nil {
		fmt.Println("❌ Test Case 1 Failed: Expected policy violation, but got success.")
		fmt.Printf("   Result ID: %v\n", result1.ID)
	} else {
		if errors.Is(err, core.ErrPolicyViolation) {
			fmt.Println("✅ Test Case 1 Passed: Request blocked as expected due to 'secret' label propagation.")
		} else {
			fmt.Printf("❌ Test Case 1 Failed: Expected ErrPolicyViolation, got: %v\n", err)
		}
	}

	// 5. Test Case 2: Input has "public" label. Output should inherit it and be allowed.
	fmt.Println("\nTest Case 2: Input with 'public' label")
	input2 := core.NewEnvelope("my public data")
	input2.AddLabel("public")

	result2, err := guardedEcho.Execute(context.Background(), input2)
	if err != nil {
		fmt.Printf("❌ Test Case 2 Failed: Expected success, got error: %v\n", err)
	} else {
		if result2.HasLabel("public") {
			fmt.Println("✅ Test Case 2 Passed: Request allowed and 'public' label propagated.")
		} else {
			fmt.Println("❌ Test Case 2 Failed: Request allowed but 'public' label NOT propagated.")
		}
	}
}
