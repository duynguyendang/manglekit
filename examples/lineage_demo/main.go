package main

import (
	"context"
	"fmt"
	"log"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/engine"
	"github.com/duynguyendang/manglekit/internal/guard"
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
	// Note: We don't need to manually propagate context here, Guard does it.
	// But wait, GuardedAction is the one that sets the context.
	// If w.Inner is a GuardedAction, it will see the parent ID set by WrapperAction's Guard.
	return w.Inner.Execute(ctx, input)
}

func (w *WrapperAction) Metadata() core.ActionMetadata {
	return core.ActionMetadata{Name: w.Name, Type: "wrapper"}
}

func main() {
	// 1. Setup Engine and Actions
	eng := engine.New()

	actionA := &MockAction{Name: "ActionA"}
	actionB := &MockAction{Name: "ActionB"}
	actionC := &MockAction{Name: "ActionC"}

	// Wrap them in Guards
	guardedA := guard.New(actionA, eng, "closed")
	guardedB := guard.New(actionB, eng, "closed")
	guardedC := guard.New(actionC, eng, "closed")

	// 2. Scenario 1: Sequential Chain (A -> B -> C)
	// We simulate a pipeline where A output is passed to B, etc.
	fmt.Println("--- Scenario 1: Sequential Chain ---")
	ctx := context.Background()

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
	guardedWrapper := guard.New(wrapper, eng, "closed")

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
	/*
		graph := eng.Lineage()

		// Print all graph facts
		facts, _ := graph.ToFacts()
		for _, f := range facts {
			fmt.Printf("Fact: %s\n", f.String())
		}

		// Check Chain: OutputC -> OutputB -> OutputA -> InputA
		parent, ok := graph.GetParent(resC.ID.String())
		if !ok {
			log.Fatalf("Missing lineage for OutputC (%s)", resC.ID)
		}
		fmt.Printf("OutputC -> %s\n", parent)
		if parent != resB.ID.String() {
			log.Fatalf("Expected OutputC -> OutputB, got -> %s", parent)
		}

		parent, ok = graph.GetParent(resB.ID.String())
		if !ok {
			log.Fatalf("Missing lineage for OutputB")
		}
		fmt.Printf("OutputB -> %s\n", parent)
		if parent != resA.ID.String() {
			log.Fatalf("Expected OutputB -> OutputA, got -> %s", parent)
		}

		parent, ok = graph.GetParent(resA.ID.String())
		if !ok {
			log.Fatalf("Missing lineage for OutputA")
		}
		fmt.Printf("OutputA -> %s\n", parent)
		if parent != inputA.ID.String() {
			log.Fatalf("Expected OutputA -> InputA, got -> %s", parent)
		}

		fmt.Println("Sequential Chain Verification PASSED")

		// Check Nested: OutputWrapper -> InputWrapper
		parent, ok = graph.GetParent(resWrapper.ID.String())
		if !ok {
			log.Fatalf("Missing lineage for OutputWrapper")
		}
		fmt.Printf("OutputWrapper -> %s\n", parent)

		if parent != inputWrapper.ID.String() {
			log.Fatalf("Expected OutputWrapper -> InputWrapper, got -> %s", parent)
		}

		fmt.Println("Nested Call Verification PASSED")
		fmt.Println("All Lineage Checks Passed!")
	*/
}
