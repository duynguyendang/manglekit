package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/engine"
	"github.com/duynguyendang/manglekit/guard"
)

// 1. Explicit Types
type EchoInput struct {
	Data string `json:"data"`
}

type EchoOutput struct {
	Result string `json:"result"`
}

// 2. Pure Logic
// EchoAction is a simple action that returns the input as output.
type EchoAction struct{}

func (a *EchoAction) Execute(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	// Simple echo: payload remains same, new envelope created
	// Note: We are using lower-level Action interface here to demonstrate Guard directly.
	// For high-level standard, we should use sdk.Define, but Taint Demo is about low-level propagation.
	// However, the rule is "Modernize all Go code".
	// Let's stick to the structure but clean up the code.

	// Payload handling
	payload, ok := input.Payload.(string)
	if !ok {
		// Try casting if it's a struct (but here we just use string for simplicity as per original demo)
		// Or modernize it to use typed payload?
		// The original used string payload.
		// If I enforce types, I should use them.
		return core.Envelope{}, fmt.Errorf("invalid payload type")
	}

	output := core.NewEnvelope(payload)
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
	policyContent := `
	Decl has_label(Req, Label).
	deny(Req) :- has_label(Req, "secret").
	`
	err := os.WriteFile("taint_policy.dlog", []byte(policyContent), 0644)
	if err != nil {
		log.Fatalf("Failed to write policy file: %v", err)
	}
	defer os.Remove("taint_policy.dlog")

	if err := eng.LoadFromPath("taint_policy.dlog"); err != nil {
		log.Fatalf("Failed to load policy: %v", err)
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
