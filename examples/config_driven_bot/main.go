package main

import (
	"context"
	"fmt"
	"log"

	"github.com/duynguyendang/manglekit/providers/google"
	"github.com/duynguyendang/manglekit/providers/openrouter"
	"github.com/duynguyendang/manglekit/sdk"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if available
	_ = godotenv.Load()

	// 1. Register Standard Plugins
	google.Register()
	openrouter.Register()

	// 2. Create Client
	// The SDK will now find the configured provider ("openrouter" or "google") in its registry.
	// We use "examples/config_driven_bot/mangle.yaml" assuming running from repo root.
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
