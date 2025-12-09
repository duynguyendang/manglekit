package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"

	"github.com/duynguyendang/manglekit/adapters/ai"
	"github.com/duynguyendang/manglekit/sdk"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"
)

func main() {
	ctx := context.Background()

	// 1. Initialize Genkit (Application Layer Responsibility)
	// access to GENIVAR or GOOGLE_API_KEY is handled by the plugin/env
	if os.Getenv("GOOGLE_API_KEY") == "" {
		fmt.Println("Warning: GOOGLE_API_KEY not set, Genkit might fail")
	}

	// Initialize the Genkit registry context
	// In the new Genkit pattern, we init the registry and plugins explicitly
	gk := genkit.Init(ctx, nil)

	// Create the model reference (Gemini 1.5 Flash is a good default)
	model := googlegenai.GoogleAIModel(gk, "gemini-1.5-flash")

	// 2. Create the Clean Adapter
	// This adapts the 'ai.Model' to 'sdk.TextGenerator'
	adapter := ai.NewGenkitAdapter(model)

	// 3. Initialize SDK with the Adapter
	// This injects the LLM into the client (Dependency Injection)
	client, err := sdk.NewClient(ctx, sdk.WithLLM(adapter))
	if err != nil {
		log.Fatal(err)
	}

	// 4. Register a Chat Action using our Adapter
	// We wrap the adapter in an LLMAction to make it a valid Manglekit Action
	chatAction, err := ai.NewLLMAction("chat", adapter)
	if err != nil {
		log.Fatal(err)
	}

	// Protect the action with policies (even if empty policy initially)
	client.RegisterAction("chat", client.Protect(chatAction))

	fmt.Println("🤖 Manglekit Simple Chat (using Explicit Genkit Adapter)")
	fmt.Println("-------------------------------------------------------")
	fmt.Println("Model: gemini-1.5-flash")
	fmt.Println("Type 'quit' or 'exit' to stop.")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("User: ")
		if !scanner.Scan() {
			break
		}
		input := scanner.Text()
		if input == "quit" || input == "exit" {
			break
		}

		// Execute the action via the Client
		// This ensures all policies and tracing run
		result, err := client.ExecuteByName(ctx, "chat", input)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		fmt.Printf("AI: %s\n\n", result.Payload)
	}
}
