package main

import (
	"context"
	"fmt"
	"os"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
)

// 1. Define Domain Types
type StockRequest struct {
	SKU string
}

type StockResponse struct {
	SKU   string
	Count int
}

// 2. Define Business Logic (Pure Go Function)
func CheckStock(ctx context.Context, req StockRequest) (StockResponse, error) {
	// Retrieve logger from context (auto-injected by guard)
	log := core.LoggerFromContext(ctx)
	log.Debug("checking stock", "sku", req.SKU)

	// Simulated business logic
	stockLevels := map[string]int{
		"IPHONE":  100,
		"MACBOOK": 25,
		"IPAD":    50,
	}

	count := stockLevels[req.SKU]
	log.Info("stock check complete", "sku", req.SKU, "count", count)
	return StockResponse{SKU: req.SKU, Count: count}, nil
}

func main() {
	ctx := context.Background()

	// 1. Initialize Manglekit with default settings (StdLogger, empty policy)
	client := manglekit.Must(manglekit.NewDefault())

	client.Logger().Info("Manglekit client initialized")

	// 2. Create a protected action in one line using ProtectFunc
	// This wraps the Go function and applies governance automatically
	protectedAction := manglekit.ProtectFunc(client, "checkStock", CheckStock)

	// 3. Execute the protected action with type-safe I/O using Call
	// No envelope packing/unpacking, no type assertions needed
	input := StockRequest{SKU: "IPHONE"}
	client.Logger().Info("Executing protected action", "sku", input.SKU)

	result, err := manglekit.Call[StockResponse](ctx, protectedAction, input)
	if err != nil {
		client.Logger().Error("protected action execution failed", "error", err)
		os.Exit(1)
	}

	// 4. Process the Result
	fmt.Println()
	fmt.Println("=== Execution Successful! ===")
	fmt.Printf("SKU: %s\n", result.SKU)
	fmt.Printf("Stock Count: %d\n", result.Count)
}
