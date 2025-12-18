package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/duynguyendang/manglekit/adapters/ai"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/logger"
	"github.com/duynguyendang/manglekit/providers/google"
	"github.com/duynguyendang/manglekit/sdk"
	"github.com/firebase/genkit/go/genkit"
	"github.com/joho/godotenv"
)

// JsonLLMAction wraps a standard LLMAction to parse output as JSON and set ContentType.
type JsonLLMAction struct {
	internal core.Action
}

func (a *JsonLLMAction) Metadata() core.ActionMetadata {
	m := a.internal.Metadata()
	m.OutputType = "map[string]string"
	return m
}

func (a *JsonLLMAction) Execute(ctx context.Context, env core.Envelope) (core.Envelope, error) {
	// 1. Delegate execution to standard LLM
	out, err := a.internal.Execute(ctx, env)
	if err != nil {
		return core.Envelope{}, err
	}

	// 2. Parse Payload (expecting string that is valid JSON)
	strPayload, ok := out.Payload.(string)
	if !ok {
		// Maybe it's already parsed?
		return out, nil
	}

	// Sanitize output (remove markdown blocks if present)
	clean := strings.TrimSpace(strPayload)
	clean = strings.TrimPrefix(clean, "```json")
	clean = strings.TrimPrefix(clean, "```")
	clean = strings.TrimSuffix(clean, "```")
	clean = strings.TrimSpace(clean)

	var parsed map[string]string
	if err := json.Unmarshal([]byte(clean), &parsed); err != nil {
		return out, fmt.Errorf("llm output not valid json: %w (Output: %s)", err, strPayload)
	}

	// 3. Update Envelope to Trigger 'TypeJSON' Flattening in Algorithm
	out.Payload = parsed
	out.ContentType = core.TypeJSON

	// Add triggering fact if needed, but flattener works on ContentType
	return out, nil
}

// GoogleJsonFactory creates the custom JSON-parsing LLM Action.
// GoogleJsonFactory creates the custom JSON-parsing LLM Action.
func GoogleJsonFactory(opts map[string]any) (sdk.ClientOption, error) {
	return func(c *sdk.Client) error {
		ctx := context.Background()

		// Initialize a local Genkit registry
		g := genkit.Init(ctx)

		apiKey := os.Getenv("GOOGLE_API_KEY")
		if apiKey == "" {
			return fmt.Errorf("GOOGLE_API_KEY environment variable is required")
		}

		modelID := "gemini-1.5-flash"
		if m, ok := opts["model"].(string); ok {
			modelID = m
		}

		name := "solve_seating"
		if n, ok := opts["action_name"].(string); ok {
			name = n
		}

		// Init Google Provider (registers model in registry)
		modelName, err := google.Init(ctx, g, apiKey, modelID)
		if err != nil {
			return fmt.Errorf("failed to init google: %w", err)
		}

		// Lookup Model
		model := genkit.LookupModel(g, modelName)
		if model == nil {
			return fmt.Errorf("model %q not found", modelName)
		}

		// Create Standard Adapter & Action
		adapter := ai.NewGenkitAdapter(model, g)
		llmAction, err := ai.NewLLMAction(name, adapter)
		if err != nil {
			return err
		}

		// Wrap in custom JSON parser
		action := &JsonLLMAction{internal: llmAction}

		// Register Action
		c.RegisterAction(name, c.Supervise(action))

		return nil
	}, nil
}

func main() {
	_ = godotenv.Load() // Load .env if present

	ctx := context.Background()

	// 1. Register Custom Provider Factory
	sdk.RegisterProvider("google", GoogleJsonFactory)

	// 2. Initialize Client from Config
	// Thishydrates "solve_seating" using our factory and loads validator.dl
	client, err := sdk.NewClientFromFile(ctx, "examples/seating_solver/mangle.yaml",
		sdk.WithLogger(logger.New("DEBUG")),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Shutdown(ctx)

	fmt.Println("--- Starting Seating Solver ---")

	// 3. Execute
	// We send a dummy prompt because the System Prompt carries the instructions.
	// We send the specific problem statement just in case.
	question := "Please solve the seating arrangement for An, Binh, Cuong, Dung."

	res, err := client.ExecuteByName(ctx, "solve_seating", question)
	if err != nil {
		log.Fatalf("Execution failed: %v", err)
	}

	// 4. Output Result
	fmt.Printf("\n>>> Final Valid Seating Arrangement: %v\n", res.Payload)
}
