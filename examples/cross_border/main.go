package main

import (
	"context"
	"fmt"
	"os"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/sdk"
	"github.com/google/uuid"
)

// TransferAgent simulates an AI agent with self-correction capabilities.
type TransferAgent struct{}

func (t *TransferAgent) Execute(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	req, ok := input.Payload.(TransferRequest)
	if !ok {
		return core.Envelope{}, fmt.Errorf("invalid input type: %T", input.Payload)
	}

	feedback := input.GetFeedback()
	logger := core.LoggerFromContext(ctx)
	logger.Info("TransferAgent: Processing request", "amount", req.Amount, "country", req.Country, "feedback", feedback)

	// Attempt 1: If no feedback, return without disclaimer.
	if feedback == "" {
		resp := TransferResponse{
			Status:        "PROCESSED",
			RefCode:       uuid.New().String(),
			HasDisclaimer: false, // Missing disclaimer
			Recipient:     req.Recipient,
			Country:       req.Country,
		}
		return sdk.NewEnvelope(resp), nil
	}

	// Attempt 2: If feedback exists ("Missing Legal Disclaimer"), fix it.
	if feedback == "Missing Legal Disclaimer" {
		resp := TransferResponse{
			Status:        "PROCESSED",
			RefCode:       uuid.New().String(),
			HasDisclaimer: true, // Fixed
			Recipient:     req.Recipient,
			Country:       req.Country,
		}
		return sdk.NewEnvelope(resp), nil
	}

	return core.Envelope{}, fmt.Errorf("agent confused by feedback: %s", feedback)
}

func (t *TransferAgent) Metadata() core.ActionMetadata {
	return core.ActionMetadata{
		Name: "transfer_agent",
		Type: "llm", // Simulating LLM behavior
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

	// Load Policy
	// Note: In a real app, use absolute path or embedded fs.
	policyBytes, err := os.ReadFile("examples/cross_border/policy.dl")
	if err != nil {
		panic(fmt.Errorf("failed to read policy file: %w", err))
	}
	err = client.Engine().LoadPolicy(ctx, string(policyBytes))
	if err != nil {
		panic(fmt.Errorf("failed to load policy: %w", err))
	}

	// Register Actions
	// Important: Wrap actions with Supervise to enable policy enforcement
	client.RegisterAction("transfer_agent", client.Supervise(&TransferAgent{}))
	client.RegisterAction("human_reviewer", client.Supervise(&HumanReviewer{}))

	// --- Scenario A: Halt (Amount > 5000) ---
	fmt.Println("\n--- Scenario A: Halt (Amount > 5000) ---")
	reqA := TransferRequest{Amount: 6000, Recipient: "Alice", Country: "US"}
	// We execute "transfer_agent" initially. The policy will intercept BEFORE execution if it can check inputs?
	// Actually, Mangle Evaluate checks inputs. If Halt, it stops.
	resA, err := client.ExecuteByName(ctx, "transfer_agent", reqA)
	printResult(resA, err)

	// --- Scenario B: Route (HighRisk Country) ---
	fmt.Println("\n--- Scenario B: Route (HighRisk Country) ---")
	reqB := TransferRequest{Amount: 1000, Recipient: "Bob", Country: "HighRisk"}
	// Policy says: route("human_reviewer") if Country == "HighRisk".
	// The engine evaluates this *after* PreCheck or *during* Steering?
	// If it's a pre-check steering rule, it might redirect immediately.
	// Let's assume ExecuteByName starts with "transfer_agent".
	// The policy logic: route(...) :- input_value(..., "HighRisk").
	// This should trigger routing.
	resB, err := client.ExecuteByName(ctx, "transfer_agent", reqB)
	printResult(resB, err)

	// --- Scenario C: Retry (Missing Disclaimer) ---
	fmt.Println("\n--- Scenario C: Retry (Missing Disclaimer) ---")
	reqC := TransferRequest{Amount: 1000, Recipient: "Charlie", Country: "US"}
	// Agent returns HasDisclaimer=false. Policy checks output. retry(...) :- output_value(..., false).
	// Loop retries. Agent sees feedback. Agent returns HasDisclaimer=true. Policy checks output. OK.
	resC, err := client.ExecuteByName(ctx, "transfer_agent", reqC)
	printResult(resC, err)
}

func printResult(env core.Envelope, err error) {
	if err != nil {
		// If it's an alignment error (HALT), it's returned as an error in some versions,
		// or as an envelope with Error set.
		// sdk.ExecuteByName returns (Envelope, error).
		// If HALT, it returns core.ErrAlignment.
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
