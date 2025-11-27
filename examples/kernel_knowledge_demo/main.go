package main

import (
	"context"
	"fmt"
	"os"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/config"
)

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
Decl user(Req, User).
Decl status(User, Status).
deny(Req) :- user(Req, U), status(U, "banned").
`
	if err := os.WriteFile("policy.dl", []byte(policyContent), 0644); err != nil {
		panic(err)
	}
	defer os.Remove("policy.dl")

	// 2. Configure Client
	cfg := &config.Config{
		Policy: config.PolicyConfig{
			Path: "policy.dl",
		},
		Knowledge: config.KnowledgeConfig{
			Path: "banned.ttl",
		},
	}

	ctx := context.Background()
	client, err := manglekit.NewClientWithConfig(ctx, cfg)
	if err != nil {
		panic(err)
	}

	// 3. Define a protected function
	type Request struct {
		User string
	}

	action := manglekit.ProtectFunc(client, "test_action", func(ctx context.Context, req Request) (string, error) {
		return "success", nil
	})

	// 4. Test Banned User
	fmt.Println("Testing banned user...")
	_, err = manglekit.Call[string](ctx, action, Request{User: "User1"})
	if err == nil {
		fmt.Println("ERROR: Expected blocked request, but it was allowed.")
		os.Exit(1)
	}
	fmt.Printf("Blocked as expected: %v\n", err)

	// 5. Test Allowed User
	fmt.Println("Testing allowed user...")
	_, err = manglekit.Call[string](ctx, action, Request{User: "User2"})
	if err != nil {
		fmt.Printf("ERROR: Expected allowed request, but it was blocked: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Allowed as expected.")
}
