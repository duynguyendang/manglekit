// Package main for 10_chatbot demonstrates the conversational capabilities
// of the Sandwich orchestrator when paired with a state provider. It simulates
// a multi-turn conversation with a fixed session ID, showing how the system
// can maintain context across multiple interactions.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/llm"
	_ "github.com/duynguyendang/manglekit/providers/all"
	"github.com/duynguyendang/manglekit/retrieve"
	"github.com/duynguyendang/manglekit/state"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	// Programmatically configure the builder with an in-memory state provider.
	builder := manglekit.NewBuilder().
		WithStateProvider(&state.InMemoryOptions{}).
		WithRetriever(&retrieve.InMemoryOptions{
			Documents: []core.Doc{
				{Text: "Mangle is a framework for building RAG applications."},
				{Text: "It is written in Go."},
			},
		}).
		WithLLM(&llm.OpenAIOptions{}) // Assumes OPENAI_API_KEY is in the environment.

	ctx := context.Background()
	pipeline, err := builder.Build(ctx)
	if err != nil {
		log.Fatalf("failed to build pipeline: %v", err)
	}
	defer pipeline.Close(ctx)

	sessionID := "chat-session-123"

	// --- Turn 1 ---
	fmt.Println("--- Turn 1 ---")
	query1 := core.Query{Text: "What is Mangle?"}
	fmt.Printf("User: %s\n", query1.Text)

	resp1, err := pipeline.Execute(ctx, sessionID, query1)
	if err != nil {
		log.Fatalf("failed to run pipeline (turn 1): %v", err)
	}
	fmt.Printf("LLM: %s\n\n", resp1.Text)

	// --- Turn 2 ---
	fmt.Println("--- Turn 2 ---")
	query2 := core.Query{Text: "What language is it written in?"}
	fmt.Printf("User: %s\n", query2.Text)

	resp2, err := pipeline.Execute(ctx, sessionID, query2)
	if err != nil {
		log.Fatalf("failed to run pipeline (turn 2): %v", err)
	}
	fmt.Printf("LLM: %s\n", resp2.Text)
}
