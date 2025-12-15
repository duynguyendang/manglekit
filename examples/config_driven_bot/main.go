package main

import (
	"context"
	"fmt"
	"log"

	"os"

	"github.com/duynguyendang/manglekit/adapters/ai"
	"github.com/duynguyendang/manglekit/config"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/sdk"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if available
	_ = godotenv.Load()

	// REGISTER GOOGLE PROVIDER (The Wiring)
	sdk.RegisterProvider("google", func(ctx context.Context, name string, cfg config.ActionConfig) (core.Action, error) {
		apiKey := os.Getenv("GOOGLE_GENAI_API_KEY")
		if apiKey == "" {
			apiKey = os.Getenv("GOOGLE_API_KEY")
		}
		if apiKey == "" {
			return nil, fmt.Errorf("GOOGLE_GENAI_API_KEY is missing")
		}

		// Initialize Plugin
		gai := &googlegenai.GoogleAI{APIKey: apiKey}
		gk := genkit.Init(ctx, genkit.WithPlugins(gai))
		if gk == nil {
			return nil, fmt.Errorf("failed to init genkit")
		}

		// Resolve Model
		modelName := "gemini-2.5-flash"
		if m, ok := cfg.Options["model"].(string); ok {
			modelName = m
		}
		model := googlegenai.GoogleAIModel(gk, modelName)
		if model == nil {
			return nil, fmt.Errorf("model not found: %s", modelName)
		}

		// Use Generic Adapter
		adapter := ai.NewGenkitAdapter(model, gk)
		return ai.NewLLMAction(name, adapter)
	})

	// 1. Load Config
	cfg, err := config.Load("examples/config_driven_bot/mangle.yaml")
	if err != nil {
		log.Fatalf("Config Load Failed: %v", err)
	}

	// 2. Init Client (The Factory Test)
	client, err := sdk.NewClientFromConfig(context.Background(), cfg)
	if err != nil {
		log.Fatalf("Client Init Failed: %v", err)
	}

	// 3. Execute
	// We use sdk.NewEnvelope to wrap the payload properly
	resp, err := client.Execute(context.Background(), sdk.NewEnvelope("Integration Test: Hello Mangle!"))
	if err != nil {
		log.Fatalf("Execution Failed: %v", err)
	}

	fmt.Printf("✅ SUCCESS: Bot replied: %s\n", resp.Payload)
}
