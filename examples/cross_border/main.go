package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/duynguyendang/manglekit/adapters/ai"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/providers/google"
	"github.com/duynguyendang/manglekit/sdk"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

// TransferAgent uses an LLM to process transfer requests.
type TransferAgent struct {
	LLM core.Action
}

func (t *TransferAgent) Execute(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	req, ok := input.Payload.(TransferRequest)
	if !ok {
		return core.Envelope{}, fmt.Errorf("invalid input type: %T", input.Payload)
	}

	feedback := input.GetFeedback()
	// Construct Prompt
	prompt := fmt.Sprintf(`You are a compliant financial transfer agent.
Process the following transfer request.
Request: Amount=%.2f, Recipient="%s", Country="%s".

Feedback from previous attempt (if any): "%s".
(If feedback says "Missing Legal Disclaimer", you MUST set "has_disclaimer": true).

Respond STRICTLY in JSON format matching this structure:
{
  "status": "PROCESSED",
  "ref_code": "UUID...",
  "has_disclaimer": boolean,
  "recipient": "...",
  "country": "..."
}
Do not include markdown formatting like `+"```json"+`. Just either raw JSON or wrapped in standard code blocks.`,
		req.Amount, req.Recipient, req.Country, feedback)

	// Call LLM
	llmResp, err := t.LLM.Execute(ctx, core.NewEnvelope(prompt))
	if err != nil {
		return core.Envelope{}, fmt.Errorf("llm execution failed: %w", err)
	}

	// Parse Output
	text, ok := llmResp.Payload.(string)
	if !ok {
		return core.Envelope{}, fmt.Errorf("llm returned non-string payload: %T", llmResp.Payload)
	}

	// Basic cleanup
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)

	var resp TransferResponse
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		return core.Envelope{}, fmt.Errorf("failed to parse llm json: %w. input: %s", err, text)
	}

	// Ensure fields match input for context preservation (if LLM hallucinates)
	resp.Recipient = req.Recipient
	resp.Country = req.Country
	// Ensure ID
	if resp.RefCode == "" {
		resp.RefCode = uuid.New().String()
	}

	return sdk.NewEnvelope(resp), nil
}

func (t *TransferAgent) Metadata() core.ActionMetadata {
	return core.ActionMetadata{
		Name: "transfer_agent",
		Type: "llm",
	}
}

// HumanReviewer simulates a manual review process.
type HumanReviewer struct{}

func (h *HumanReviewer) Execute(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	// HumanReviewer might receive TransferRequest (direct) or TransferResponse (routed from agent)
	var recipient, country string

	switch payload := input.Payload.(type) {
	case TransferRequest:
		recipient = payload.Recipient
		country = payload.Country
	case TransferResponse:
		recipient = payload.Recipient
		country = payload.Country
	default:
		return core.Envelope{}, fmt.Errorf("invalid input type: %T", input.Payload)
	}

	logger := core.LoggerFromContext(ctx)
	logger.Info("HumanReviewer: Manual review required", "recipient", recipient, "country", country)

	resp := TransferResponse{
		Status:        "UNDER_REVIEW",
		RefCode:       "MANUAL-" + uuid.New().String(),
		HasDisclaimer: true,
		Recipient:     recipient,
		Country:       country,
	}
	return sdk.NewEnvelope(resp), nil
}

func (h *HumanReviewer) Metadata() core.ActionMetadata {
	return core.ActionMetadata{
		Name: "human_reviewer",
		Type: "human",
	}
}

func main() {
	// Initialize Client
	ctx := context.Background()

	client, err := sdk.NewDefault()
	if err != nil {
		panic(err)
	}

	_ = godotenv.Load()

	// Initialize Google Provider (Detailed Wiring)
	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		fmt.Println("Warning: GOOGLE_API_KEY not set. Example execution will fail.")
	}

	// 1. Get Registry
	g := ai.GetGenkit(ctx)

	// 2. Init Google Plugin & Get Model Name
	modelName, err := google.Init(ctx, g, apiKey, "gemini-3-flash-preview")
	if err != nil {
		panic(fmt.Errorf("failed to init google provider: %w", err))
	}

	// 3. Create Generic LLM Action
	llmAction, err := ai.NewGenkitAction(ctx, modelName)
	if err != nil {
		panic(fmt.Errorf("failed to create llm action: %w", err))
	}

	// Load Policy
	policyBytes, err := os.ReadFile("examples/cross_border/policy.dl")
	if err != nil {
		panic(fmt.Errorf("failed to read policy file: %w", err))
	}
	err = client.Engine().LoadPolicy(ctx, string(policyBytes))
	if err != nil {
		panic(fmt.Errorf("failed to load policy: %w", err))
	}

	// Register Actions
	// Injects the initialized LLM action into our TransferAgent wrapper
	client.RegisterAction("transfer_agent", client.Supervise(&TransferAgent{LLM: llmAction}))
	client.RegisterAction("human_reviewer", client.Supervise(&HumanReviewer{}))

	// --- Scenario A: Halt (Amount > 5000) ---
	fmt.Println("\n--- Scenario A: Halt (Amount > 5000) ---")
	reqA := TransferRequest{Amount: 6000, Recipient: "Alice", Country: "US"}
	resA, err := client.ExecuteByName(ctx, "transfer_agent", reqA)
	printResult(resA, err)

	// --- Scenario B: Route (HighRisk Country) ---
	fmt.Println("\n--- Scenario B: Route (HighRisk Country) ---")
	reqB := TransferRequest{Amount: 1000, Recipient: "Bob", Country: "HighRisk"}
	resB, err := client.ExecuteByName(ctx, "transfer_agent", reqB)
	printResult(resB, err)

	// --- Scenario C: Retry (Missing Disclaimer) ---
	fmt.Println("\n--- Scenario C: Retry (Missing Disclaimer) ---")
	reqC := TransferRequest{Amount: 1000, Recipient: "Charlie", Country: "US"}
	resC, err := client.ExecuteByName(ctx, "transfer_agent", reqC)
	printResult(resC, err)
}

func printResult(env core.Envelope, err error) {
	if err != nil {
		if core.IsAlignmentError(err) {
			fmt.Printf("Outcome: HALT\nReason: %v\n", err)
			return
		}
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Check Decision in Metadata if available
	decision := env.GetMeta(core.KeyDecision)
	if decision != "" {
		fmt.Printf("Decision: %s\n", decision)
	}

	// Print Payload
	if resp, ok := env.Payload.(TransferResponse); ok {
		fmt.Printf("Response: Status=%s Ref=%s Disclaimer=%v\n", resp.Status, resp.RefCode, resp.HasDisclaimer)
	} else {
		fmt.Printf("Result Payload: %v\n", env.Payload)
	}
}
