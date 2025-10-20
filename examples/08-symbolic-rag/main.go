package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/providers/all"
	"github.com/duynguyendang/manglekit/sdk"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	ctx := context.Background()

	// Read the configuration file.
	configData, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}

	// Create a new registry and register all the standard providers.
	registry := manglekit.NewRegistry()
	all.Register(registry)

	// Load the orchestrator from the configuration data.
	orch, err := sdk.FromConfig(ctx, registry, configData)
	if err != nil {
		log.Fatalf("Failed to build orchestrator: %v", err)
	}
	defer orch.Close(ctx)

	// Execute a query.
	query := core.Query{
		Text: "What is Manglekit?",
	}

	answer, err := orch.Execute(ctx, "session-123", query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Execution failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Answer:", answer.Text)
}
