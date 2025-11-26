package main

import (
	"context"
	"fmt"
	"log"

	"github.com/duynguyendang/manglekit/v2/adapters/func"
	"github.com/duynguyendang/manglekit/v2/core"
	"github.com/duynguyendang/manglekit/v2/engine"
	"github.com/duynguyendang/manglekit/v2/guard"
)

// 1. Define Domain
type StockReq struct {
	SKU string
}

type StockResp struct {
	Count int
}

// 2. Define Logic
func CheckStock(ctx context.Context, req StockReq) (StockResp, error) {
	log.Printf("Checking stock for SKU: %s", req.SKU)
	// Dummy logic
	if req.SKU == "IPHONE" {
		return StockResp{Count: 100}, nil
	}
	return StockResp{Count: 0}, nil
}

func main() {
	ctx := context.Background()

	// 3. Wiring
	// Wrap the Go function into a core.Action
	checkStockAction := function.New("checkStock", CheckStock)

	// Create a policy engine (no rules for this demo)
	policyEngine := engine.New()

	// Wrap the action with the security guard
	guardedAction := guard.New(checkStockAction, policyEngine)

	// 4. Execution
	// Create an input envelope
	inputEnvelope := core.NewEnvelope(StockReq{SKU: "IPHONE"})

	log.Println("Executing guarded action...")
	outputEnvelope, err := guardedAction.Execute(ctx, inputEnvelope)
	if err != nil {
		log.Fatalf("guarded action execution failed: %v", err)
	}

	// 5. Verify
	result, ok := outputEnvelope.Payload.(StockResp)
	if !ok {
		log.Fatalf("invalid output payload type: got %T", outputEnvelope.Payload)
	}

	fmt.Println("Execution successful!")
	fmt.Printf("Stock for SKU 'IPHONE': %d\n", result.Count)
}
