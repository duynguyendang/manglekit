package main

import (
	"context"
	"fmt"
	"time"

	"github.com/duynguyendang/manglekit/examples/proposalgpt/adapters"
	"github.com/duynguyendang/manglekit/internal/adapters/mangle"
	"github.com/duynguyendang/manglekit/internal/audit"
	"github.com/duynguyendang/manglekit/internal/core/domain"
	"github.com/duynguyendang/manglekit/internal/core/logic"
	"github.com/duynguyendang/manglekit/internal/genepool"
	"github.com/duynguyendang/manglekit/internal/orchestrator"
)

func main() {
	fmt.Println("===========================================")
	fmt.Println(" Manglekit v2 Sovereign Kernel (Mock Demo) ")
	fmt.Println("===========================================")

	ctx := context.Background()

	// 1. Initialize Adapters
	llmAdapter := adapters.NewMockGenerative()
	storage := adapters.NewMockStorage("/tmp/manglekit")

	// 2. Initialize Internal Subsystems
	reasoner := mangle.NewReasoningAdapter()
	auditorLayer := audit.New(reasoner)
	promptCompiler := logic.NewCompiler()

	// 3. Boot-Time Integrity Check (GenePool)
	activePool, err := genepool.New(ctx, storage, "mock_manifest")
	if err != nil {
		fmt.Printf("Boot Failure: %v\n", err)
		return
	}

	// 4. Initialize the OODA State Machine (Orchestrator)
	kernel := orchestrator.New(
		adapters.NewConsolePerception(),
		llmAdapter,
		promptCompiler,
		auditorLayer,
		activePool,
		storage,
	)

	// 5. Trigger an Epoch
	userRequest := domain.Signal{
		ID:         "sig-001",
		Timestamp:  time.Now(),
		Intent:     "DeployDatabase",
		RawContent: "Please deploy the production postgres instance.",
	}

	fmt.Printf("\n[OBSERVE] Ingesting Signal %s (Intent: %s)\n", userRequest.ID, userRequest.Intent)

	result, err := kernel.Execute(ctx, userRequest)

	if err != nil {
		fmt.Printf("\n[DEADLOCK] Cognitive Execution Failed: %v\n", err)
	} else {
		presenter := adapters.NewConsolePresentation()
		_ = presenter.Render(ctx, *result)
	}
}
