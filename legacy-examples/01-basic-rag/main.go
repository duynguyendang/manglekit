package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/config"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/providers/all"
	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

func main() {
	godotenv.Load()
	ctx := context.Background()

	// Read the configuration file.
	configData, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}

	// Create a new registry.
	registry := manglekit.NewRegistry()

	// Parse the config to get the orchestrator name.
	var cfg config.Config
	if err := yaml.Unmarshal(configData, &cfg); err != nil {
		log.Fatalf("Error unmarshalling config: %v", err)
	}

	// Create a new builder and load from config.
	builder := manglekit.NewBuilder(registry).WithHandlers(all.ComponentHandlers()...)
	orch, _, err := builder.FromConfig(ctx, configData)
	if err != nil {
		log.Fatalf("Error building orchestrator: %v", err)
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
