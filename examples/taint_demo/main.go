package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
)

// 2. Pure Logic
// EchoAction is a simple action that returns the input as output.
type EchoAction struct{}

func (a *EchoAction) Execute(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	// Simple echo: payload remains same, new envelope created
	// Payload handling
	payload, ok := input.Payload.(string)
	if !ok {
		return core.Envelope{}, fmt.Errorf("invalid payload type")
	}

	output := core.NewEnvelope(payload)
	// Note: Taint propagation is handled by the Guard, not the Action.
	// The Action just does work.
	return output, nil
}

func (a *EchoAction) Metadata() core.ActionMetadata {
	return core.ActionMetadata{
		Name: "Echo",
		Type: "Utility",
	}
}

func main() {
	ctx := context.Background()

	// 2. Define Policy: Block output if it has "secret" label
	policyContent := `
	Decl has_label(Req, Label).
	deny(Req) :- has_label(Req, "secret").
	`
	err := os.WriteFile("taint_blueprint.dlog", []byte(policyContent), 0644)
	if err != nil {
		log.Fatalf("Failed to write policy file: %v", err)
	}
	defer os.Remove("taint_blueprint.dlog")

	// 1. Setup Client with Blueprint (replaces direct Engine usage)
	client := manglekit.Must(manglekit.NewClient(
		ctx,
		manglekit.WithBlueprintPath("taint_blueprint.dlog"),
		manglekit.WithFailMode("closed"),
	))

	// 3. Register Action
	echo := &EchoAction{}
	// Use client.RegisterAction to attach the action to the client (and thus the guard)
	// Note: client.RegisterAction wraps the action with Protect() automatically.
	client.RegisterAction("echo", echo)

	// 4. Test Case 1: Input has "secret" label. Output should inherit it and be blocked.
	fmt.Println("Test Case 1: Input with 'secret' label")
	input1 := core.NewEnvelope("my secret data")
	input1.AddLabel("secret")

	// We use client.ExecuteEnvelope directly to pass labels manually (simulating upstream taint)
	// ExecuteByName takes (ctx, name, payload). It doesn't allow setting labels easily on the *input* envelope
	// because it creates a NEW envelope from payload.
	// To test Taint Propagation from *Input Envelope*, we need to bypass `ExecuteByName` or use `sdk.WithEnvelopeOption`?
	// `manglekit` doesn't expose `ExecuteEnvelope`.

	// However, `client` has `Protect(action)`.
	// If we use `Protect`, we get a `core.Action` back.
	// We can run `Execute` on that guarded action with our pre-labeled envelope.

	guardedEcho := client.Supervise(echo)

	result1, err := guardedEcho.Execute(context.Background(), input1)
	if err == nil {
		fmt.Println("❌ Test Case 1 Failed: Expected blueprint alignment issue, but got success.")
		fmt.Printf("   Result ID: %v\n", result1.ID)
	} else {
		// core.ErrAlignment might be wrapped
		if errors.Is(err, core.ErrAlignment) || strings.Contains(err.Error(), "alignment issue") {
			fmt.Println("✅ Test Case 1 Passed: Request blocked as expected due to 'secret' label propagation.")
		} else {
			// Check for wrapped error
			var pve *core.AlignmentError
			if errors.As(err, &pve) {
				fmt.Println("✅ Test Case 1 Passed: Request blocked as expected due to 'secret' label propagation.")
			} else {
				fmt.Printf("❌ Test Case 1 Failed: Expected ErrAlignment, got: %v\n", err)
			}
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
