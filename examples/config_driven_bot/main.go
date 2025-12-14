package main

import (
	"context"
	"fmt"
	"log"

	"github.com/duynguyendang/manglekit/config"
	"github.com/duynguyendang/manglekit/sdk"
)

func main() {
    // 1. Load Config
	cfg, err := config.Load("examples/config_driven_bot/mangle.yaml")
	if err != nil {
		log.Fatalf("Config Load Failed: %v", err)
	}

    // 2. Init Client (The Factory Test)
	client, err := sdk.NewClientFromConfig(context.Background(), cfg)
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
