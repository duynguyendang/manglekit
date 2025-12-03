package main

import (
	"context"
	"fmt"
	"log"

	"github.com/duynguyendang/manglekit/adapters/extractor"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/engine"
)

type Order struct {
	Product  string `json:"product" mangle:"product"`
	Quantity int    `json:"qty" mangle:"qty"`
	Urgent   bool   `json:"urgent" mangle:"urgent"`
}

type MockLLM struct{}

func (m *MockLLM) Execute(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	// Pure function logic
	return core.NewEnvelope(`{"product": "Laptop", "qty": 1, "urgent": true}`), nil
}

func (m *MockLLM) Metadata() core.ActionMetadata {
	return core.ActionMetadata{Name: "mock-llm", Type: "llm"}
}

func main() {
	llm := &MockLLM{}

	// Init extractor
	// Extractor returns an Action.
	// We use it via Execute currently as it's an Adapter.
	// To modernize, we might wrap it in sdk.Define?
	// But ExtractorAction is already an Action.
	// We just ensure type safety on the result.

	ext, err := extractor.New("order_parser", llm, Order{})
	if err != nil {
		log.Fatalf("Failed to create extractor: %v", err)
	}

	input := core.NewEnvelope("I need a Laptop ASAP, just one.")

	// Execute
	// We could use sdk.Call if we had a wrapped function, but here we use the Action directly.
	// This fits "Invisible Governance" if we Protect it?
	// The example demonstrates "Extractor" capability.
	// I will keep direct Execute but clean up logging.

	result, err := ext.Execute(context.Background(), input)
	if err != nil {
		log.Fatalf("Execution failed: %v", err)
	}

	order, ok := result.Payload.(Order)
	if !ok {
		log.Fatalf("Payload is not Order, got %T", result.Payload)
	}

	fmt.Printf("Extracted Struct: %+v\n", order)

	// Verification
	if order.Product != "Laptop" {
		log.Fatalf("Expected Product 'Laptop', got '%s'", order.Product)
	}

	// Engine Reflection Demo
	fmt.Println("Generating Mangle Facts...")

	facts, err := engine.ToFacts(result.ID.String(), order)
	if err != nil {
		log.Fatalf("Fact generation failed: %v", err)
	}

	for _, f := range facts {
		fmt.Printf("Fact: %s\n", f)
	}
}
