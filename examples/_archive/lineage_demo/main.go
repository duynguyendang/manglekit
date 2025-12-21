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
	log := core.LoggerFromContext(ctx)
	log.Info("Executing action", "name", m.Name, "input_id", input.ID.String())
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
	log := core.LoggerFromContext(ctx)
	log.Info("Executing nested action", "name", w.Name, "input_id", input.ID.String())
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
	supervisedA := client.Supervise(actionA)
	supervisedB := client.Supervise(actionB)
	supervisedC := client.Supervise(actionC)

	// 2. Scenario 1: Sequential Chain (A -> B -> C)
	// We simulate a pipeline where A output is passed to B, etc.
	fmt.Println("--- Scenario 1: Sequential Chain ---")

	// A
	inputA := core.NewEnvelope("Secret Data")
	fmt.Printf("InputA ID: %s\n", inputA.ID)
	resA, err := supervisedA.Execute(ctx, inputA)
	if err != nil {
		log.Fatalf("A failed: %v", err)
	}
	fmt.Printf("OutputA ID: %s (Derived from InputA)\n", resA.ID)

	// B
	resB, err := supervisedB.Execute(ctx, resA)
	if err != nil {
		log.Fatalf("B failed: %v", err)
	}
	fmt.Printf("OutputB ID: %s (Derived from OutputA)\n", resB.ID)

	// C
	resC, err := supervisedC.Execute(ctx, resB)
	if err != nil {
		log.Fatalf("C failed: %v", err)
	}
	fmt.Printf("OutputC ID: %s (Derived from OutputB)\n", resC.ID)

	// 3. Scenario 2: Nested Calls (Wrapper -> Inner)
	fmt.Println("\n--- Scenario 2: Nested Calls ---")

	// Create a wrapper that calls C
	wrapper := &WrapperAction{Name: "Wrapper", Inner: supervisedC}
	// Protect the wrapper too
	supervisedWrapper := client.Supervise(wrapper)

	inputWrapper := core.NewEnvelope("Nested Data")
	fmt.Printf("InputWrapper ID: %s\n", inputWrapper.ID)

	resWrapper, err := supervisedWrapper.Execute(ctx, inputWrapper)
	if err != nil {
		log.Fatalf("Wrapper failed: %v", err)
	}
	fmt.Printf("OutputWrapper ID: %s\n", resWrapper.ID)

	// 4. Verification
	fmt.Println("\n--- Verification ---")
	fmt.Println("Lineage verification via in-memory graph is deprecated in favor of OpenTelemetry tracing.")
	// Original code commented out graph checks. We leave it at that.
}
