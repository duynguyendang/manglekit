package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/sdk"
)

// 1. Explicit Types
type HistoryReq struct {
	Message string `json:"message"`
}

type HistoryRes struct {
	Count int    `json:"count"`
	Note  string `json:"note"`
}

// 2. Pure Logic
type HistoryCheckAction struct {
	Retried bool
}

func (a *HistoryCheckAction) Execute(ctx context.Context, env core.Envelope) (core.Envelope, error) {
	// Check history
	historyJSON := env.Metadata[core.KeyHistory]
	var history []core.ChatMessage
	if historyJSON != "" {
		_ = json.Unmarshal([]byte(historyJSON), &history)
	}

	// Force one retry if not already retried
	decision := core.DecisionAllow
	if !a.Retried {
		a.Retried = true
		decision = core.DecisionRetry // Force a retry
	}

	res := core.NewEnvelope(HistoryRes{
		Count: len(history),
		Note:  fmt.Sprintf("History Count: %d", len(history)),
	})

	// Inject steering decision
	// core.Envelope.Metadata is map[string]string.
	// core.DecisionAllow/Retry are strings ("ALLOW", "RETRY").
	if res.Metadata == nil {
		res.Metadata = make(map[string]string)
	}
	res.Metadata[core.KeyDecision] = decision
	res.Metadata[core.KeyFeedback] = "forced_retry"

	return res, nil
}

func (a *HistoryCheckAction) Metadata() core.ActionMetadata {
	return core.ActionMetadata{Name: "check_history", Type: "checker"}
}

func main() {
	ctx := context.Background()
	client, err := sdk.NewDefault()
	if err != nil {
		log.Fatalf("Error initializing client: %v", err)
	}
	defer client.Shutdown(ctx)

	// Case 1: Stateless
	fmt.Println("--- Case 1: Stateless (None) ---")
	action1 := &HistoryCheckAction{}
	client.RegisterAction("check_history", action1)

	res1, err := client.ExecuteByName(ctx, "check_history", HistoryReq{Message: "hello"})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Final Result: %v\n", res1.Payload)
	}

	// Case 2: Transient
	fmt.Println("\n--- Case 2: Transient ---")
	action2 := &HistoryCheckAction{}
	client.RegisterAction("check_history_transient", action2)

	res2, err := client.ExecuteByName(ctx, "check_history_transient", HistoryReq{Message: "hello"}, sdk.WithTransientMemory())
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Final Result: %v\n", res2.Payload)
	}
}
