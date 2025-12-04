package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
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
// NOTE: Stateful actions in loop (like Retried bool) are generally bad practice in "Pure Logic"
// but for this demo proving the loop mechanism, we keep it manually in the struct.
// However, sdk.Runnable/Define enforces functional purity.
// To use standard `Define`, we can't easily maintain state *inside* the handler struct if it's recreated.
// But `Define` takes a closure. So closure capture works.

func main() {
	ctx := context.Background()
	// 1. Init Client
	client := manglekit.Must(manglekit.NewClient(ctx))
	defer client.Shutdown(ctx)

	// Case 1: Stateless
	fmt.Println("--- Case 1: Stateless (None) ---")

	action1 := &HistoryCheckAction{}
	client.RegisterAction("check_history", action1)

	// ExecuteByName is available on client.
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

	// Use manglekit options
	res2, err := client.ExecuteByName(ctx, "check_history_transient", HistoryReq{Message: "hello"}, manglekit.WithTransientMemory())
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Final Result: %v\n", res2.Payload)
	}
}

// 2. Raw Action Implementation (needed for advanced control flow)
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

	if res.Metadata == nil {
		res.Metadata = make(map[string]string)
	}
	res.Metadata[core.KeyDecision] = decision
	res.Metadata[core.KeyPrevFeedback] = "forced_retry" // Use correct key

	return res, nil
}

func (a *HistoryCheckAction) Metadata() core.ActionMetadata {
	return core.ActionMetadata{Name: "check_history", Type: "checker"}
}
