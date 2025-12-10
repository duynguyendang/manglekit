package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
)

// Input struct for routing
type Input struct {
	Tier string `mangle:"payload.tier"`
	User string `mangle:"payload.user"`
}

// SQLOutput struct for generation
type SQLOutput struct {
	SQL string `mangle:"payload.sql"`
}

func main() {
	ctx := context.Background()

	// 1. Initialize Client
	client := manglekit.Must(manglekit.NewClient(ctx, manglekit.WithBlueprintPath("examples/steering/blueprint.dl")))

	// 2. Register Actions

	// Action 1: SQL Generator (demonstrates RETRY)
	gen := &SQLGenerator{}
	client.RegisterAction("generate_sql", client.Protect(gen))

	// Action 2: Router (demonstrates ROUTE)
	client.RegisterAction("classify", client.Protect(&RouterAction{}))

	// Action 3: VIP Agent
	client.RegisterAction("vip_agent", client.Protect(&VIPAction{}))

	// 3. RunLoop - Scenario 1: Retry
	fmt.Println("--- Scenario 1: Retry (Bad SQL) ---")
	res, err := client.ExecuteByName(ctx, "generate_sql", nil)
	if err != nil {
		fmt.Printf("RunLoop failed: %v\n", err)
	} else {
		// Extract SQL from payload
		if out, ok := res.Payload.(SQLOutput); ok {
			fmt.Printf("Result: %s\n", out.SQL)
		} else {
			fmt.Printf("Result: %+v\n", res.Payload)
		}
	}

	// 3. RunLoop - Scenario 2: Route
	fmt.Println("\n--- Scenario 2: Route (Gold Tier) ---")
	// Note: We use Input struct which maps to payload.tier
	res, err = client.ExecuteByName(ctx, "classify", Input{Tier: "gold"})
	if err != nil {
		fmt.Printf("RunLoop failed: %v\n", err)
	} else {
		if val, ok := res.Payload.(string); ok {
			fmt.Printf("Result: %s\n", val)
		} else {
			fmt.Printf("Result: %+v\n", res.Payload)
		}
	}
}

// SQLGenerator implementation
type SQLGenerator struct{}

func (a *SQLGenerator) Execute(ctx context.Context, env core.Envelope) (core.Envelope, error) {
	// Check previous feedback
	feedback := env.Metadata[core.KeyPrevFeedback]

	sql := "SELECT * FROM users; DROP TABLE users;" // Default bad
	if strings.Contains(feedback, "Do not use DROP") {
		sql = "SELECT * FROM users; DELETE FROM users;" // Fixed
	}

	// Create output envelope with SQLOutput payload
	return core.NewEnvelope(SQLOutput{SQL: sql}), nil
}

func (a *SQLGenerator) Metadata() core.ActionMetadata {
	return core.ActionMetadata{Name: "generate_sql", Type: "generator"}
}

// RouterAction implementation
type RouterAction struct{}

func (a *RouterAction) Execute(ctx context.Context, env core.Envelope) (core.Envelope, error) {
	// Pass through the input as output
	return env, nil
}
func (a *RouterAction) Metadata() core.ActionMetadata {
	return core.ActionMetadata{Name: "classify", Type: "router"}
}

// VIPAction implementation
type VIPAction struct{}

func (a *VIPAction) Execute(ctx context.Context, env core.Envelope) (core.Envelope, error) {
	return core.NewEnvelope("VIP Service Executed"), nil
}
func (a *VIPAction) Metadata() core.ActionMetadata {
	return core.ActionMetadata{Name: "vip_agent", Type: "agent"}
}
