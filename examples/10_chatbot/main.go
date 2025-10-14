// Package main for 10_chatbot demonstrates how to build a stateful chatbot
// using Manglekit's StateProvider feature.
//
// This example simulates a multi-turn conversation where the chatbot's memory
// (conversation history) is persisted across multiple calls using the in-memory
// state provider. It shows how to retrieve the history for a given session,
// append new messages, and save it back, enabling the LLM to have context of
// the ongoing dialogue.
package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	_ "github.com/duynguyendang/manglekit/providers/all"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	ctx := context.Background()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatalf("failed to get current file path")
	}
	dir := filepath.Dir(currentFile)
	configPath := filepath.Join(dir, "config.yaml")

	builder, err := manglekit.NewBuilderFromYAML(configPath)
	if err != nil {
		log.Fatalf("Failed to create builder from YAML: %v", err)
	}

	orch, err := builder.Build(ctx)
	if err != nil {
		log.Fatalf("Failed to build orchestrator: %v", err)
	}
	defer orch.Close(ctx)

	sessionID := uuid.NewString()
	fmt.Printf("Starting new chat session: %s\n", sessionID)
	fmt.Println("Ask a question, or type 'exit' to quit.")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}
		userInput := scanner.Text()
		if strings.ToLower(userInput) == "exit" {
			break
		}

		// 1. Retrieve the conversation history from the state provider.
		history, err := orch.StateProvider().Get(ctx, sessionID)
		if err != nil {
			log.Printf("Error getting state: %v", err)
			continue
		}
		var historySlice []string
		if history != nil {
			// When retrieving from JSON, it will be []any, so we need to convert.
			if historySliceAny, ok := history.([]any); ok {
				for _, item := range historySliceAny {
					if str, ok := item.(string); ok {
						historySlice = append(historySlice, str)
					}
				}
			} else if concreteSlice, ok := history.([]string); ok {
				historySlice = concreteSlice
			} else {
				log.Printf("Error: state is not a slice of strings or []any, but %T", history)
				continue
			}
		}

		// 2. Augment the user's query with the history.
		var prompt strings.Builder
		prompt.WriteString("Conversation History:\n")
		for _, msg := range historySlice {
			prompt.WriteString(msg)
			prompt.WriteString("\n")
		}
		prompt.WriteString("\nNew Question: ")
		prompt.WriteString(userInput)

		query := core.Query{Text: prompt.String()}

		// 3. Execute the pipeline.
		answer, err := orch.Execute(ctx, sessionID, query)
		if err != nil {
			log.Printf("Pipeline execution failed: %v", err)
			continue
		}

		fmt.Println("Bot:", answer.Text)

		// 4. Update the history and save it back to the state provider.
		historySlice = append(historySlice, "User: "+userInput)
		historySlice = append(historySlice, "Bot: "+answer.Text)
		if err := orch.StateProvider().Set(ctx, sessionID, historySlice); err != nil {
			log.Printf("Error setting state: %v", err)
		}
	}
}
