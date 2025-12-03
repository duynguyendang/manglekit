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
	// In a real scenario, we would parse the prompt to ensure it's correct.
	// Here we just return the expected JSON.
	return core.NewEnvelope(`{"product": "Laptop", "qty": 1, "urgent": true}`), nil
}

func (m *MockLLM) Metadata() core.ActionMetadata {
	return core.ActionMetadata{Name: "mock-llm", Type: "llm"}
}

func main() {
	llm := &MockLLM{}

	// Init extractor
	// We pass Order{} which means we expect Order struct back.
	ext, err := extractor.New("order_parser", llm, Order{})
	if err != nil {
		log.Fatalf("Failed to create extractor: %v", err)
	}

	input := core.NewEnvelope("I need a Laptop ASAP, just one.")

	fmt.Println("Executing Extractor...")
	result, err := ext.Execute(context.Background(), input)
	if err != nil {
		log.Fatalf("Execution failed: %v", err)
	}

	order, ok := result.Payload.(Order)
	if !ok {
		log.Fatalf("Payload is not Order, got %T", result.Payload)
	}

	fmt.Printf("Extracted Struct: %+v\n", order)

	// Verification Logic
	if order.Product != "Laptop" {
		log.Fatalf("Expected Product 'Laptop', got '%s'", order.Product)
	}
	if order.Quantity != 1 {
		log.Fatalf("Expected Quantity 1, got %d", order.Quantity)
	}
	if !order.Urgent {
		log.Fatalf("Expected Urgent true, got false")
	}

	fmt.Println("Struct Verification Successful!")

	// Demonstrate Engine Reflection
	// Note: We need a valid entity ID for facts. The Envelope has an ID.
	fmt.Println("Generating Mangle Facts...")

	facts, err := engine.ToFacts(result.ID.String(), order)
	if err != nil {
		log.Fatalf("Fact generation failed: %v", err)
	}

	for _, f := range facts {
		fmt.Printf("Fact: %s\n", f)
	}
}
