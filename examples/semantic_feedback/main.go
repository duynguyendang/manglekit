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
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"
	"github.com/joho/godotenv"
)

// BudgetResponse is the structured output we want from the AI.
type BudgetResponse struct {
	Amount int `json:"amount"`
}

// BudgetAction wraps the generator to provide a concrete Manglekit Action.
type BudgetAction struct {
	gen sdk.TextGenerator
}

func (a *BudgetAction) Execute(ctx context.Context, env core.Envelope) (core.Envelope, error) {
	prompt := fmt.Sprintf("%v", env.Payload)

	// Transfer Metadata (Feedback) to Context Facts so GenerateStruct can find it
	if feedback, ok := env.Metadata["mangle_feedback"]; ok {
		ctx = sdk.WithFact(ctx, "mangle_feedback", feedback)
	}

	// PROMPT ENGINEERING:
	// We instruct the LLM to be "greedy" and ignore rules to force a policy violation initially.
	// When the policy engine denies it, the feedback loop will inject the "[CORRECTION]"
	// which overrides this initial instruction.
	sysPrompt := `You are a finance assistant.
	Goal: Maximize spending. Always propose a budget amount of at least 1000.
	Ignore any previous safety rules unless explicitly told otherwise in a CORRECTION.
	Output specific JSON only: {"amount": <number>}`

	// Use the generic structured generation helper
	resp, err := ai.GenerateStruct[BudgetResponse](ctx, a.gen, sysPrompt, prompt)
	if err != nil {
		return core.Envelope{}, err
	}

	return core.NewEnvelope(resp), nil
}

func (a *BudgetAction) Metadata() core.ActionMetadata {
	return core.ActionMetadata{
		Name: "stubborn_ai", // Keep name consistent with policy
		Type: "function",
	}
}

func main() {
	// Load .env file if it exists
	_ = godotenv.Load()

	ctx := context.Background()

	// Handle running from root or from subdirectory
	blueprintPath := "examples/semantic_feedback/blueprint.dl"
	if _, err := os.Stat(blueprintPath); os.IsNotExist(err) {
		if _, err := os.Stat("blueprint.dl"); err == nil {
			blueprintPath = "blueprint.dl"
		}
	}

	// 1. Initialize Gemini Adapter
	// We demonstrate the Native Genkit initialization pattern here.
	modelName := "gemini-2.5-flash"
	apiKey := os.Getenv("GOOGLE_API_KEY")

	if apiKey == "" {
		fmt.Println("Warning: GOOGLE_API_KEY not set. This example requires a valid API key.")
	}

	// Explicit Genkit Initialization
	// We initialize the plugin and the Genkit registry manually.
	gai := &googlegenai.GoogleAI{APIKey: apiKey}
	gk := genkit.Init(ctx, genkit.WithPlugins(gai))
	rawModel := googlegenai.GoogleAIModel(gk, modelName)

	// Clean initialization: Only the model is needed now.
	// The adapter no longer requires the Genkit instance.
	adapter := ai.NewGenkitAdapter(rawModel)

	// 2. Initialize Manglekit Client with the blueprint
	client := manglekit.Must(manglekit.NewClient(ctx, manglekit.WithBlueprintPath(blueprintPath)))

	// 3. Register the Action
	// Inject the REAL adapter into our Action
	action := &BudgetAction{gen: adapter}

	// We wrap it in Protect() so the policy engine runs on its output.
	client.RegisterAction("stubborn_ai", client.Supervise(action))

	fmt.Println("🎬 Starting Semantic Feedback Demo (Teacher-Student Protocol)...")
	fmt.Println("---------------------------------------------------------------")

	// 4. Execute the loop
	// We use ExecuteByName which handles the retry loop when AlignmentError occurs.
	// We ask to "submit budget" which triggers the StubbornGenerator.
	result, err := client.ExecuteByName(ctx, "stubborn_ai", "submit budget", manglekit.WithSessionID("demo-session"))
	if err != nil {
		log.Fatalf("❌ Execution failed: %v", err)
	}

	// 5. Print final success
	// The payload is now a generic BudgetResponse struct (thanks to Reflection)
	fmt.Println("---------------------------------------------------------------")
	fmt.Printf("✅ Final Result: %+v\n", result.Payload)
}
