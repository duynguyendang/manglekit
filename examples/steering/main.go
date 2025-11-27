package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
)

// MockAction is a simple action that returns its input as output.
type MockAction struct {
	name string
}

func (a *MockAction) Execute(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	fmt.Printf("[Action: %s] Executing with payload: %v\n", a.name, input.Payload)
	// For testing, just return the input payload as output
	return core.NewEnvelope(input.Payload), nil
}

func (a *MockAction) Metadata() core.ActionMetadata {
	return core.ActionMetadata{Name: a.name}
}

// SQLGenAction simulates generating SQL.
type SQLGenAction struct {
	name string
}

func (a *SQLGenAction) Execute(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	fmt.Printf("[Action: %s] Generating SQL...\n", a.name)

	// Check for feedback
	if fb, ok := input.Metadata["prev_feedback"]; ok {
		fmt.Printf("[Action: %s] Received feedback: %s\n", a.name, fb)
		// If feedback says "Do not use DROP", fix it.
		if strings.Contains(fb, "Do not use DROP") {
			return core.NewEnvelope(map[string]string{"sql": "DELETE FROM users"}), nil
		}
	}

	// Default behavior: Generate dangerous SQL
	return core.NewEnvelope(map[string]string{"sql": "DROP TABLE users"}), nil
}

func (a *SQLGenAction) Metadata() core.ActionMetadata {
	return core.ActionMetadata{Name: a.name}
}

func main() {
	ctx := context.Background()

	// 1. Initialize Client with Steering Policy
	client, err := manglekit.NewClient(ctx, "examples/steering/policy.dl")
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// 2. Register Actions
	// Standard SQL Generator
	client.RegisterAction("sql_gen", &SQLGenAction{name: "sql_gen"})

	// VIP Agent
	client.RegisterAction("vip_agent", &MockAction{name: "vip_agent"})

	// Router Entry Point
	client.RegisterAction("router", &MockAction{name: "router"})

	// 3. Test Case 1: Retry Logic (Self-Correction)
	fmt.Println("\n--- Test Case 1: Retry (Self-Correction) ---")
	// Input doesn't matter much for this mock, but let's start with sql_gen
	res, err := client.RunLoop(ctx, "sql_gen", map[string]string{"instruction": "delete users"})
	if err != nil {
		log.Fatalf("RunLoop failed: %v", err)
	}
	fmt.Printf("Final Result: %v\n", res.Payload)

	// 4. Test Case 2: Routing Logic (Dynamic Dispatch)
	fmt.Println("\n--- Test Case 2: Route (VIP) ---")
	// Start with "router". Input has tier="gold".
	// The policy says: next_step("Req", "vip_agent") :- payload.tier("Req", "gold").

	res, err = client.RunLoop(ctx, "router", map[string]string{"tier": "gold"})
	if err != nil {
		log.Fatalf("RunLoop failed: %v", err)
	}
	fmt.Printf("Final Result: %v\n", res.Payload)
}
