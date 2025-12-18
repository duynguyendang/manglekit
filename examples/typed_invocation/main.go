package main

import (
	"context"
	"fmt"
	"log"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/sdk"
)

// --- 1. Define Data Models ---

type UserProfileReq struct {
	UserID string `json:"user_id"`
}

type UserProfileResp struct {
	Name string `json:"name"`
	Role string `json:"role"`
}

// --- 2. Define Action Handle (Single Source of Truth) ---

var ActionGetUser = sdk.DefineAction[UserProfileReq, UserProfileResp]("get_user_profile")

// --- 3. Mock Action Implementation ---

type MockAction struct {
	Func func(ctx context.Context, env core.Envelope) (core.Envelope, error)
}

func (m *MockAction) Execute(ctx context.Context, env core.Envelope) (core.Envelope, error) {
	return m.Func(ctx, env)
}

func (m *MockAction) Metadata() core.ActionMetadata {
	return core.ActionMetadata{Name: "mock"}
}

func main() {
	ctx := context.Background()
	client, _ := sdk.NewDefault()

	// Register a mock implementation (Pro-Code style)
	// Note: In real life, this might come from a plugin or config.
	client.RegisterAction("get_user_profile", &MockAction{
		Func: func(ctx context.Context, env core.Envelope) (core.Envelope, error) {
			// Simulate returning a generic map (like from a DB or API)
			return core.NewEnvelope(map[string]any{
				"name": "Alice",
				"role": "Admin",
			}), nil
		},
	})

	// --- 3. Execute with Type Safety ---

	req := UserProfileReq{UserID: "u-123"}

	// ERROR: Compiler fails if you pass wrong struct
	// resp, err := sdk.Execute(ctx, client, ActionGetUser, "wrong-input")

	// SUCCESS: Correct types
	resp, err := sdk.Execute(ctx, client, ActionGetUser, req)
	if err != nil {
		log.Fatal(err)
	}

	// resp is automatically UserProfileResp
	fmt.Printf("User: %s, Role: %s\n", resp.Name, resp.Role)
}
