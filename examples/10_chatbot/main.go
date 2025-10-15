package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/google/uuid"
)

func main() {
	ctx := context.Background()
	// This example uses the default in-memory state provider.
	// For production, you would configure a persistent provider (e.g., Redis).
	builder, err := manglekit.NewBuilderFromYAML("examples/10_chatbot/testdata/config.yaml")
	if err != nil {
		log.Fatalf("Error initializing builder: %v", err)
	}

	orchestrator, err := builder.Build(ctx)
	if err != nil {
		log.Fatalf("Error building orchestrator: %v", err)
	}
	defer orchestrator.Close(ctx)

	// Generate a unique session ID for this conversation.
	sessionID := uuid.NewString()

	fmt.Println("Chatbot session started. Type 'exit' to end.")
	fmt.Println("=============================================")

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("You: ")
		userInput, _ := reader.ReadString('\n')
		userInput = strings.TrimSpace(userInput)

		if strings.ToLower(userInput) == "exit" {
			fmt.Println("Chatbot session ended.")
			break
		}

		if userInput == "" {
			continue
		}

		query := core.Query{Text: userInput}
		answer, err := orchestrator.Execute(ctx, sessionID, query)
		if err != nil {
			log.Printf("Error executing query: %v", err)
			continue
		}

		fmt.Printf("Bot: %s\n", answer.Text)
	}
}
