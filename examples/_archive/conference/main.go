package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/sdk"
	"github.com/joho/godotenv"

	// Import Google provider to ensure it's registered
	_ "github.com/duynguyendang/manglekit/providers/google"
)

// Data Structures for JSON Parsing
type ExtractedData struct {
	Speakers    []Speaker    `json:"speakers"`
	Preferences []Preference `json:"preferences"`
	Conflicts   []Conflict   `json:"conflicts"`
	Rooms       []Room       `json:"rooms"`
	Assignments []Assignment `json:"assignments"`
}

type Speaker struct {
	Name     string   `json:"name"`
	Requires []string `json:"requires"`
}

type Preference struct {
	Speaker string `json:"speaker"`
	Slot    string `json:"slot"`
}

type Conflict struct {
	S1 string `json:"s1"`
	S2 string `json:"s2"`
}

type Room struct {
	Name     string   `json:"name"`
	Features []string `json:"features"`
}

type Assignment struct {
	Speaker string `json:"speaker"`
	Room    string `json:"room"`
	Slot    string `json:"slot"`
}

// ValidatorAction is a dummy action to trigger the Governance Loop validation
type ValidatorAction struct{}

func (v *ValidatorAction) Execute(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	// If we reached here, Pre-check passed.
	return core.NewEnvelope("Schedule Verified"), nil
}

func (v *ValidatorAction) Metadata() core.ActionMetadata {
	return core.ActionMetadata{Name: "validate_schedule"}
}

// --- Mock Support for Verification ---

type MockGenerator struct {
	Response string
}

func (m *MockGenerator) Generate(ctx context.Context, prompt string, opts ...core.GenerateOption) (*core.LLMResponse, error) {
	return &core.LLMResponse{Text: m.Response}, nil
}
func (m *MockGenerator) Complete(ctx context.Context, prompt string) (string, error) {
	return m.Response, nil
}
func (m *MockGenerator) Stream(ctx context.Context, prompt string) (<-chan string, error) {
	return nil, fmt.Errorf("not implemented")
}

func init() {
	sdk.RegisterProvider("mock", func(opts map[string]any) (sdk.ClientOption, error) {
		return func(c *sdk.Client) error {
			name, _ := opts["_action_name"].(string)
			prompt, _ := opts["prompt"].(string)

			// We use the 'prompt' option in config as the Mock Response for this test
			// We use the 'prompt' option in config as the Mock Response for this test

			// We need to construct an action. Use a simple wrapper or if adapters/ai is available
			// But for simplicity, we can implement core.Action interface on a struct that uses MockGenerator
			// Or check if we can import adapters/ai

			// For this example, let's just register a dummy action that returns the text
			action := &MockAction{Name: name, Response: prompt}
			c.RegisterAction(name, action)
			return nil
		}, nil
	})
}

type MockAction struct {
	Name     string
	Response string
}

func (m *MockAction) Execute(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	return core.NewEnvelope(m.Response), nil
}
func (m *MockAction) Metadata() core.ActionMetadata {
	return core.ActionMetadata{Name: m.Name}
}

func main() {
	_ = godotenv.Overload()
	// 1. Setup
	if os.Getenv("GOOGLE_API_KEY") == "" {
		fmt.Println("Please set GOOGLE_API_KEY")
		return // Graceful exit for CI/Test without keys
	}
	ctx := context.Background()

	// 2. Initialize Client
	configPath := "examples/conference/mangle.yaml"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}
	client, err := sdk.NewClientFromFile(ctx, configPath)
	if err != nil {
		log.Fatalf("Failed to init client: %v", err)
	}

	// 3. User Input
	// Hardcoded for this example
	inputPrompt := "Speaker A hates B. A needs a projector. B wants to talk in the morning. Room 101 has no projector. Room 102 has a projector."
	fmt.Printf("📝 Input: %s\n", inputPrompt)

	// 4. Extraction & Validation Loop
	const MaxRetries = 3
	var extractionHistory []string // Keep track of prompt + previous failures
	currentPrompt := inputPrompt

	for attempt := 1; attempt <= MaxRetries; attempt++ {
		fmt.Printf("\n🔄 Attempt %d/%d\n", attempt, MaxRetries)

		// 4a. Extraction Phase
		fmt.Println("🧠 Extracting facts & proposing schedule...")

		// If we have history (feedback), we should append it to the prompt or use a chat-like structure.
		// For simplicity, we append it to the prompt string here.
		// In a real app, we'd use sdk.WithHistory() or similar if the action supported chat mode history.
		// But here "fact_extractor" is LLM type, so it takes string input.

		promptToUse := currentPrompt
		if len(extractionHistory) > 0 {
			promptToUse += "\n\nPREVIOUS ATTEMPTS & FEEDBACK:\n" + strings.Join(extractionHistory, "\n")
		}

		resp, err := client.ExecuteByName(ctx, "fact_extractor", promptToUse)
		if err != nil {
			log.Fatalf("Extraction failed: %v", err)
		}

		// 5. Parse JSON
		var data ExtractedData
		jsonStr := fmt.Sprintf("%v", resp.Payload)
		// Clean up potential markdown blocks
		jsonStr = strings.TrimPrefix(jsonStr, "```json")
		jsonStr = strings.TrimSuffix(jsonStr, "```")
		// Also handle if lines are wrapped in code blocks with language specifier
		if idx := strings.Index(jsonStr, "{"); idx != -1 {
			jsonStr = jsonStr[idx:]
		}
		if idx := strings.LastIndex(jsonStr, "}"); idx != -1 {
			jsonStr = jsonStr[:idx+1]
		}

		if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
			log.Printf("Failed to parse JSON: %v\nRaw: %s", err, jsonStr)
			// This is a failure we can feed back too!
			feedback := fmt.Sprintf("Error parsing JSON: %v. Please output valid JSON only.", err)
			extractionHistory = append(extractionHistory, feedback)
			fmt.Printf("⚠️ Feedback: %s\n", feedback)
			continue
		}

		// 6. Convert to Facts
		var facts []string
		// Always clear facts? Engine state is persistent in this client instance?
		// No, `client.LoadFacts` appends to the store.
		// Manglekit Engine usually accumulates facts.
		// If we loop, we might be adding duplicate facts or conflicting facts from previous attempts if we reuse the same client/engine.
		// For this example, we should probably reset the facts or use a fresh client/engine scope, or rely on Datalog handling duplicates (sets).
		// However, old assignments will persist.
		// To fix this, we need to retract old facts. The SDK doesn't support Retract easily yet.
		// WORKAROUND: Re-initialize the engine/facts for each attempt is safest for this example without 'retract'.
		// But we want to reuse the client configuration.
		// We can just create a `client.Engine().ClearFacts()` if it existed, or just reload everything.
		// BUT `LoadFacts` is additive.

		// Let's assume for this example we are just testing the *Validation logic* on the *current dataset*.
		// If we re-run, we might pollute the state.
		// Actually, let's just create a NEW client for each attempt to be clean, or use `client.Engine().Reset()` if available.
		// Checking SDK... `sdk/client.go` -> `engine` is `core.Evaluator`.
		// Usually no Reset.
		// Okay, let's re-init client inside the loop or just ignore duplication if Datalog handles it (it usually ignores duplicate ground facts).
		// Problem: Old assignments A->101 and New assignments A->102 will BOTH exist, causing conflict!
		// Solution: We MUST clear state.
		// Re-creating client is the easiest way here.

		// For the loop to work with client re-creation, we move client init inside or create a distinct scope.
		// Let's move the `NewClientFromFile` call inside the loop (or just the fact loading part if we could clear).
		// Re-init client slightly expensive but safe.

		loopClient, err := sdk.NewClientFromFile(ctx, configPath)
		if err != nil {
			log.Fatalf("Failed to re-init client: %v", err)
		}

		// ... (Facts Mapping Loop)
		for _, s := range data.Speakers {
			facts = append(facts, fmt.Sprintf("speaker(%q).", s.Name))
			for _, req := range s.Requires {
				facts = append(facts, fmt.Sprintf("requires(%q, %q).", s.Name, req))
			}
		}
		for _, p := range data.Preferences {
			facts = append(facts, fmt.Sprintf("preference(%q, %q).", p.Speaker, p.Slot))
		}
		for _, c := range data.Conflicts {
			facts = append(facts, fmt.Sprintf("conflict(%q, %q).", c.S1, c.S2))
			facts = append(facts, fmt.Sprintf("conflict(%q, %q).", c.S2, c.S1))
		}
		for _, r := range data.Rooms {
			facts = append(facts, fmt.Sprintf("room(%q).", r.Name))
			for _, f := range r.Features {
				facts = append(facts, fmt.Sprintf("room_feature(%q, %q).", r.Name, f))
			}
		}
		for _, a := range data.Assignments {
			facts = append(facts, fmt.Sprintf("assignment(%q, %q, %q).", a.Speaker, a.Room, a.Slot))
		}

		// 7. Inject Facts into Engine
		if err := loopClient.LoadFacts(facts); err != nil {
			log.Fatalf("Failed to load facts: %v", err)
		}
		fmt.Printf("✅ Loaded %d facts.\n", len(facts))

		// 7b. Load Policy (Manually, so facts exist)
		policyContent, err := os.ReadFile("examples/conference/policy.dl")
		if err != nil {
			log.Fatalf("Failed to read policy: %v", err)
		}
		if err := loopClient.Engine().LoadPolicy(ctx, string(policyContent)); err != nil {
			log.Fatalf("Failed to load policy: %v", err)
		}

		// 8. Governance Loop (Verification)
		validator := &ValidatorAction{}
		supervisedValidator := loopClient.Supervise(validator)

		_, err = supervisedValidator.Execute(ctx, core.NewEnvelope("check"))

		// DEMO: Simulate failure on first attempt to show the loop
		if attempt == 1 && err == nil {
			err = fmt.Errorf("Simulated Logic Error: Speaker A needs projector but 101 has none")
		}

		if err != nil {
			fmt.Printf("❌ Validation Result: HALTED / RETRY\n")
			fmt.Printf("Reason: %v\n", err)

			// CAPTURE FEEDBACK
			feedback := fmt.Sprintf("Constraint Violation: %v. Please fix the schedule.", err)
			extractionHistory = append(extractionHistory, feedback)
			fmt.Printf("🔄 Feeding back error to LLM...\n")

			// Wait a bit not to hammer API
			// time.Sleep(1 * time.Second)
			continue
		} else {
			fmt.Printf("✅ Validation Result: VALID\n")
			for _, a := range data.Assignments {
				fmt.Printf("- %s in %s at %s\n", a.Speaker, a.Room, a.Slot)
			}
			return // Success!
		}
	}
	fmt.Println("❌ Max retries exceeded. Could not find a valid schedule.")
}
