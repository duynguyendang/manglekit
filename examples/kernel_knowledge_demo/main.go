package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/config"
	"github.com/duynguyendang/manglekit/sdk"
)

// 1. Explicit Types
type Request struct {
	User string `mangle:"request_user"`
}

type Response struct {
	Status string
}

// 2. Pure Logic
func testAction(ctx context.Context, req Request) (Response, error) {
	return Response{Status: "success"}, nil
}

func main() {
	// 1. Create temporary policy and knowledge files
	ttlContent := `
@prefix ex: <http://ex/> .
ex:User1 ex:status "banned" .
`
	if err := os.WriteFile("banned.ttl", []byte(ttlContent), 0644); err != nil {
		panic(err)
	}
	defer os.Remove("banned.ttl")

	policyContent := `
Decl request_user(Req, User).
Decl status(User, Status).
deny(Req) :- request_user(Req, U), status(U, "banned").
`
	if err := os.WriteFile("policy.dl", []byte(policyContent), 0644); err != nil {
		panic(err)
	}
	defer os.Remove("policy.dl")

	// 2. Configure Client
	// Note: We use sdk.NewClientWithConfig here because the simplified Facade
	// does not currently expose configuration for Knowledge Base paths.
	cfg := &config.Config{
		Policy: config.PolicyConfig{
			Path: "policy.dl",
		},
		Knowledge: config.KnowledgeConfig{
			Path: "banned.ttl",
		},
	}

	ctx := context.Background()
	client, err := sdk.NewClientWithConfig(ctx, cfg)
	if err != nil {
		log.Fatalf("Client init failed: %v", err)
	}
	defer client.Shutdown(ctx)

	// 3. Generic Binding using Facade
	var TestAction = manglekit.Define(client, "test_action", testAction)

	// 4. Test Banned User
	fmt.Println("Testing banned user...")
	_, err = TestAction.Run(ctx, Request{User: "User1"})
	if err == nil {
		fmt.Println("ERROR: Expected blocked request, but it was allowed.")
		os.Exit(1)
	} else {
		fmt.Printf("Blocked as expected: %v\n", err)
	}

	// 5. Test Allowed User
	fmt.Println("Testing allowed user...")
	_, err = TestAction.Run(ctx, Request{User: "User2"})
	if err != nil {
		fmt.Printf("ERROR: Expected allowed request, but it was blocked: %v\n", err)
		os.Exit(1)
	} else {
		fmt.Println("Allowed as expected.")
	}
}
