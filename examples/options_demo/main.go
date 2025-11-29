package main

import (
	"context"
	"fmt"
	"log"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
)

func main() {
	ctx := context.Background()

	// 1. Clean Init
	client, err := manglekit.NewClient(ctx,
		manglekit.WithPolicyPath("examples/options_demo/policy.dl"),
		manglekit.WithFailMode("open"),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Register a dummy action for "chat_agent"
	client.RegisterAction("chat_agent", &dummyAction{})

	// 2. Clean Execution
	res, err := client.ExecuteByName(ctx, "chat_agent", "Hello",
		manglekit.WithSessionID("user-123"),
		manglekit.WithMetadata("source", "mobile"),
	)
	if err != nil {
		log.Printf("Execution failed: %v", err)
	} else {
		fmt.Printf("Result: %v\n", res.Payload)
	}
}

type dummyAction struct{}

func (a *dummyAction) Execute(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	fmt.Printf("Executing chat_agent with input: %v\n", input.Payload)
	fmt.Printf("Metadata: %v\n", input.Metadata)
	return core.NewEnvelope("Hello back!"), nil
}

func (a *dummyAction) Metadata() core.ActionMetadata {
	return core.ActionMetadata{Name: "chat_agent"}
}
