package main

import (
	"context"
	"fmt"
	"log"

	"github.com/duynguyendang/manglekit/providers/google"
	"github.com/duynguyendang/manglekit/sdk"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if available
	_ = godotenv.Load()

	// 1. Register Standard Plugins
	google.Register()

	// 2. Create Client
	// The SDK will now find "google" in its registry automatically.
	// We use the path relative to repo root as per previous example behavior,
	// or "mangle.yaml" if running from the directory.
    // The previous code used "examples/config_driven_bot/mangle.yaml".
	client, err := sdk.NewClientFromFile(context.Background(), "examples/config_driven_bot/mangle.yaml")
	if err != nil {
		log.Fatalf("Client Init Failed: %v", err)
	}

	// 3. Execute
	// We use sdk.NewEnvelope to wrap the payload properly
	resp, err := client.Execute(context.Background(), sdk.NewEnvelope("Integration Test: Hello Mangle!"))
	if err != nil {
		log.Fatalf("Execution Failed: %v", err)
	}

	fmt.Printf("✅ SUCCESS: Bot replied: %s\n", resp.Payload)
}
