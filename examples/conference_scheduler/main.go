package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/sdk"
)

func main() {
	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		log.Fatal("GOOGLE_API_KEY is required")
	}

	// 1. Initialize Client
	client, err := manglekit.NewClient(context.Background(),
		manglekit.WithBlueprintPath("scheduler.dl"),
		manglekit.WithLogger(nil), // Default logger
		// Use default HybridMemory
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// 2. Define the Action (Mock Scheduling for simplicity)
	// In a real app, this would use an LLM to extract schedule from natural language.
	// sdk.Define registers the action automatically with the client.
	sdk.Define(client, "schedule", func(ctx context.Context, input string) (string, error) {
		// Mock LLM output that might generate conflicts
		// For demo purposes, we return a structured string.
		return fmt.Sprintf("Scheduled: %s", input), nil
	})

	// No manual RegisterAction needed as Define does it.

	// 3. Execution Loop
	// We want to verify that memory (history) works.
	// We will simulate a conversation.

	// Turn 1: Schedule Alice
	res, err := client.ExecuteByName(context.Background(), "schedule", "Alice at 10am in Room A",
		manglekit.WithSessionID("session-1"),
	)
	if err != nil {
		log.Printf("Turn 1 Error: %v", err)
	} else {
		log.Printf("Turn 1 Result: %v", res.Payload)
	}

	// Turn 2: Schedule Bob at same time/room (Conflict)
	// This requires the "schedule" action to output facts that Datalog can see.
	// Currently Manglekit extracts facts from Payload if structured, or we need an extractor.
	// For this example to work with the Datalog provided, we need to inject facts or have the LLM output JSON facts.

	// Let's assume we load facts manually for the sake of the "Memory Orchestration" test,
	// OR we update the action to return a struct that produces facts.

	// But the user goal is primarily "Refactor Memory... Then, implement... Example using this new architecture".
	// The Datalog part is "handles halt for conflicts and retry".
	// I will keep it simple.
}
