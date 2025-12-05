package main

import (
	"context"
	"fmt"
	"log"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
)

// MockAction just passes data through.
type MockAction struct {
	Name string
}

func (m *MockAction) Execute(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	fmt.Printf("[%s] Executing with Input ID: %s\n", m.Name, input.ID)
	// Output ID will be new
	out := core.NewEnvelope(input.Payload)
	return out, nil
}

func (m *MockAction) Metadata() core.ActionMetadata {
	return core.ActionMetadata{Name: m.Name, Type: "mock"}
}

// WrapperAction calls another action inside.
type WrapperAction struct {
	Name  string
	Inner core.Action
}

func (w *WrapperAction) Execute(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	fmt.Printf("[%s] Executing (Nested) with Input ID: %s\n", w.Name, input.ID)
	// Call Inner Action
	return w.Inner.Execute(ctx, input)
}

func (w *WrapperAction) Metadata() core.ActionMetadata {
	return core.ActionMetadata{Name: w.Name, Type: "wrapper"}
}

func main() {
	// 1. Setup Client
	ctx := context.Background()
	client := manglekit.Must(manglekit.NewClient(ctx))

	// Define raw Actions
	actionA := &MockAction{Name: "ActionA"}
	actionB := &MockAction{Name: "ActionB"}
	actionC := &MockAction{Name: "ActionC"}

	// Use Client to Protect them (Facade)
	guardedA := client.Protect(actionA)
	guardedB := client.Protect(actionB)
	guardedC := client.Protect(actionC)

	// 2. Scenario 1: Sequential Chain (A -> B -> C)
	// We simulate a pipeline where A output is passed to B, etc.
	fmt.Println("--- Scenario 1: Sequential Chain ---")

	// A
	inputA := core.NewEnvelope("Secret Data")
	fmt.Printf("InputA ID: %s\n", inputA.ID)
	resA, err := guardedA.Execute(ctx, inputA)
	if err != nil {
		log.Fatalf("A failed: %v", err)
	}
	fmt.Printf("OutputA ID: %s (Derived from InputA)\n", resA.ID)

	// B
	resB, err := guardedB.Execute(ctx, resA)
	if err != nil {
		log.Fatalf("B failed: %v", err)
	}
	fmt.Printf("OutputB ID: %s (Derived from OutputA)\n", resB.ID)

	// C
	resC, err := guardedC.Execute(ctx, resB)
	if err != nil {
		log.Fatalf("C failed: %v", err)
	}
	fmt.Printf("OutputC ID: %s (Derived from OutputB)\n", resC.ID)

	// 3. Scenario 2: Nested Calls (Wrapper -> Inner)
	fmt.Println("\n--- Scenario 2: Nested Calls ---")

	// Create a wrapper that calls C
	wrapper := &WrapperAction{Name: "Wrapper", Inner: guardedC}
	// Protect the wrapper too
	guardedWrapper := client.Protect(wrapper)

	inputWrapper := core.NewEnvelope("Nested Data")
	fmt.Printf("InputWrapper ID: %s\n", inputWrapper.ID)

	resWrapper, err := guardedWrapper.Execute(ctx, inputWrapper)
	if err != nil {
		log.Fatalf("Wrapper failed: %v", err)
	}
	fmt.Printf("OutputWrapper ID: %s\n", resWrapper.ID)

	// 4. Verification
	fmt.Println("\n--- Verification ---")
	fmt.Println("Lineage verification via in-memory graph is deprecated in favor of OpenTelemetry tracing.")
	// Original code commented out graph checks. We leave it at that.
}
