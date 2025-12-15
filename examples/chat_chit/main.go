package main

import (
	"context"
	"fmt"
	"log"
	"os"

	mangleai "github.com/duynguyendang/manglekit/adapters/ai"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/providers/google"
	"github.com/duynguyendang/manglekit/sdk"
	"github.com/joho/godotenv"
)

// AnalysisResult matches the JSON structure we want from the LLM
type AnalysisResult struct {
	Topic     string `json:"topic"`     // "billing", "technical", "general"
	Sentiment string `json:"sentiment"` // "positive", "negative", "neutral"
}

// IntentExtractorAction is the "Perception Layer" (Right Brain).
// It uses Genkit (LLM) to convert unstructured text -> structured intent.
type IntentExtractorAction struct {
	llm core.TextGenerator
}

func (a *IntentExtractorAction) Metadata() core.ActionMetadata {
	return core.ActionMetadata{
		Name:      "extract_intent",
		InputType: "string",
		Type:      "perception",
	}
}

func (a *IntentExtractorAction) Execute(ctx context.Context, env core.Envelope) (core.Envelope, error) {
	stmt, ok := env.Payload.(string)
	if !ok {
		return core.Envelope{}, fmt.Errorf("expected string payload")
	}

	fmt.Printf("[Perception] Reading user input: %q\n", stmt)

	// --- STEP A: GENKIT (THE MUSCLE) ---
	// "Analyze the user input. Classify topic (billing/technical/general) and sentiment (negative/neutral/positive)."
	sysPrompt := "You are an intent classifier. Output specific JSON only: {\"topic\": \"...\", \"sentiment\": \"...\"}. " +
		"Topics: billing, technical, general_inquiry. Sentiment: negative, neutral, positive."

	analysis, err := mangleai.GenerateStruct[AnalysisResult](
		ctx,
		a.llm,
		sysPrompt,
		stmt,
	)
	if err != nil {
		return core.Envelope{}, fmt.Errorf("extraction failed: %w", err)
	}

	fmt.Printf("[Perception] Extracted Intent: %+v\n", analysis)

	// --- STEP B: FACT CONVERSION (THE BRIDGE) ---
	// Convert Genkit's struct into Mangle's Datalog facts.
	facts := []string{
		fmt.Sprintf("topic(%q)", analysis.Topic),
		fmt.Sprintf("sentiment(%q)", analysis.Sentiment),
	}

	// --- STEP C: PACK THE ENVELOPE ---
	// Pass the original payload (cargo) + facts (label)
	out := core.NewEnvelope(stmt)
	out.Facts = facts

	// Add derived_from metadata for tracing
	if env.ID.String() != "" {
		out.SetMeta("derived_from", env.ID.String())
	}

	return out, nil
}

// ---------------------------------------------------------
// The three destination bots
// ---------------------------------------------------------

// 1. Human Manager (Billing + Angry)
type HumanManagerAction struct{}

func (h *HumanManagerAction) Metadata() core.ActionMetadata {
	return core.ActionMetadata{Name: "human_manager", Type: "human_handoff"}
}
func (h *HumanManagerAction) Execute(ctx context.Context, env core.Envelope) (core.Envelope, error) {
	fmt.Printf("\n>>> [Human Manager] ALARM! I will handle this refund personally.\n")
	return core.NewEnvelope("Refund processed by human"), nil
}

// 2. Tech Support Bot (Technical + Neutral)
type TechSupportAction struct{}

func (t *TechSupportAction) Metadata() core.ActionMetadata {
	return core.ActionMetadata{Name: "tech_support_bot", Type: "model"}
}
func (t *TechSupportAction) Execute(ctx context.Context, env core.Envelope) (core.Envelope, error) {
	fmt.Printf("\n>>> [Tech Support] Have you tried turning it off and on again?\n")
	return core.NewEnvelope("Standard technical response"), nil
}

// 3. General Chat Bot (Default)
type GeneralBotAction struct{}

func (g *GeneralBotAction) Metadata() core.ActionMetadata {
	return core.ActionMetadata{Name: "general_chat_bot", Type: "model"}
}
func (g *GeneralBotAction) Execute(ctx context.Context, env core.Envelope) (core.Envelope, error) {
	fmt.Printf("\n>>> [General Bot] How can I help you today?\n")
	return core.NewEnvelope("General greeting"), nil
}

// ---------------------------------------------------------
// Main
// ---------------------------------------------------------

func main() {
	ctx := context.Background()
	_ = godotenv.Overload()

	// 1. Initialize Client with Protocol (Logic)
	protocolPath := "examples/chat_chit/protocol.dl"
	cwd, _ := os.Getwd()
	fmt.Printf("Running from: %s\n", cwd)

	client, err := sdk.NewClient(ctx, sdk.WithBlueprintPath(protocolPath), sdk.WithFailMode("closed"))
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// 2. Initialize Genkit (Perception) using Google Provider
	// We use the new provider 'google' package to get the generator
	if os.Getenv("GOOGLE_API_KEY") == "" {
		log.Fatal("GOOGLE_API_KEY is not set")
	}

	// Create the Manglekit TextGenerator adapter via provider
	gemini, err := google.New(ctx, google.WithModel("gemini-2.5-flash"))
	if err != nil {
		log.Fatalf("Failed to initialize google provider: %v", err)
	}

	// 3. Register Actions
	intentExtractor := &IntentExtractorAction{llm: gemini}

	client.RegisterAction("extract_intent", client.Supervise(intentExtractor))
	client.RegisterAction("human_manager", client.Supervise(&HumanManagerAction{}))
	client.RegisterAction("tech_support_bot", client.Supervise(&TechSupportAction{}))
	client.RegisterAction("general_chat_bot", client.Supervise(&GeneralBotAction{}))

	// 4. Run Scenarios
	runScenario(ctx, client, "Angry Billing", "I was charged twice! Refund me NOW!")
	runScenario(ctx, client, "Calm Technical", "My screen generates a weird flickering noise when I open the app.")
	runScenario(ctx, client, "Unknown / General", "Is it going to rain today?")
}

func runScenario(ctx context.Context, client *sdk.Client, name, input string) {
	fmt.Printf("\n--- Scenario: %s ---\n", name)

	// ExecuteByName runs the loop.
	// 1. extract_intent runs -> returns facts.
	// 2. Engine evaluates facts -> Decisions ROUTE to X.
	// 3. Loop runs X.
	res, err := client.ExecuteByName(ctx, "extract_intent", input)
	if err != nil {
		log.Printf("Scenario failed: %v", err)
		return
	}
	fmt.Printf("Final Result Payload: %v\n", res.Payload)
}
