package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/adapters/ai"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/sdk"
	genkitai "github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/compat_oai/openai"
	"github.com/joho/godotenv"
	"github.com/openai/openai-go/option"
)

// SimpleChatAction wraps the generator to provide a concrete Manglekit Action.
type SimpleChatAction struct {
	gen sdk.TextGenerator
}

func (a *SimpleChatAction) Execute(ctx context.Context, env core.Envelope) (core.Envelope, error) {
	prompt := fmt.Sprintf("%v", env.Payload)

	// Define a simple structured response
	type ChatResponse struct {
		Reply string `json:"reply"`
	}

	// Use GenerateStruct for structured output
	sysPrompt := "You are a helpful assistant."
	resp, err := ai.GenerateStruct[ChatResponse](ctx, a.gen, sysPrompt, prompt)
	if err != nil {
		return core.Envelope{}, err
	}

	return core.NewEnvelope(resp), nil
}

func (a *SimpleChatAction) Metadata() core.ActionMetadata {
	return core.ActionMetadata{
		Name: "chat_action",
		Type: "function",
	}
}

func main() {
	_ = godotenv.Load()
	ctx := context.Background()

	// 1. Configuration
	apiKey := os.Getenv("OPENAI_API_KEY")
	baseURL := os.Getenv("OPENAI_BASE_URL")
	modelName := "mistralai/mistral-7b-instruct:free" // Example OpenRouter model

	if apiKey == "" {
		fmt.Println("Warning: OPENAI_API_KEY not set. Please set it to run this example.")
		// Providing a dummy key to prevent panic during initialization if just testing setup
		apiKey = "dummy-key-for-init"
	}

	// 2. Initialize OpenAI Plugin with Custom Base URL
	oai := &openai.OpenAI{APIKey: apiKey}
	if baseURL != "" {
		fmt.Printf("Using Custom Base URL: %s\n", baseURL)
		oai.Opts = []option.RequestOption{option.WithBaseURL(baseURL)}
	}

	// 3. Initialize Genkit
	gk := genkit.Init(ctx, genkit.WithPlugins(oai))

	// 4. Define/Get the Model
	// Since OpenRouter has many models, we often need to define it dynamically if not pre-registered.
	rawModel := oai.Model(gk, modelName)
	if rawModel == nil {
		fmt.Printf("Model %s not found, registering dynamically...\n", modelName)
		rawModel = oai.DefineModel(modelName, genkitai.ModelOptions{
			Label: modelName,
			Supports: &genkitai.ModelSupports{
				Multiturn:  true,
				SystemRole: true,
			},
		})
	}

	// 5. Create Adapter
	adapter := ai.NewGenkitAdapter(rawModel, gk)

	// 6. Setup Manglekit Client
	blueprintPath := "examples/openrouter_demo/blueprint.dl"
	if _, err := os.Stat(blueprintPath); os.IsNotExist(err) {
		blueprintPath = "blueprint.dl" // Fallback if running from dir
	}

	client := manglekit.Must(manglekit.NewClient(ctx, manglekit.WithBlueprintPath(blueprintPath)))

	// 7. Register Action
	action := &SimpleChatAction{gen: adapter}
	client.RegisterAction("chat", client.Supervise(action))

	// 8. Execute
	fmt.Println("🤖 Sending request to", modelName, "via", baseURL)
	res, err := client.ExecuteByName(ctx, "chat", "Hello, are you running on OpenRouter?", manglekit.WithSessionID("test-session"))
	if err != nil {
		// If we don't have a valid key, this will fail, which is expected.
		// We print the error clearly.
		log.Printf("Execution failed (likely due to missing/invalid key): %v", err)
		return
	}

	fmt.Printf("✅ Reply: %+v\n", res.Payload)
}
