package main

import (
	"context"
	"fmt"
	"log"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/adapters/extractor"
	"github.com/duynguyendang/manglekit/core"
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
	// 1. Init Client (Standard)
	client := manglekit.Must(manglekit.NewClient(context.Background()))

	// 2. Define Components
	llm := &MockLLM{}

	ext, err := extractor.New("order_parser", llm, Order{})
	if err != nil {
		log.Fatalf("Failed to create extractor: %v", err)
	}

	// 3. Protect Action via Facade (Required for Governance)
	// We wrap the extractor action with client.Protect to ensure all policies are enforced.
	guardedExt := client.Protect(ext)

	// Execute
	input := core.NewEnvelope("I need a Laptop ASAP, just one.")
	result, err := guardedExt.Execute(context.Background(), input)
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

	fmt.Println("Extraction successful.")
}
