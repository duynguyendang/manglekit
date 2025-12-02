package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/sdk"
)

// InventoryDB simulates Redis
var InventoryDB map[string]int

// fn:get_inventory implementation in Go
func get_inventory(productID string) (int, bool) {
	qty, ok := InventoryDB[productID]
	return qty, ok
}

// PricingRequest represents the input for the pricing agent
type PricingRequest struct {
	User      string `mangle:"request_user"`
	Product   string `mangle:"request_product"`
	IsVIP     bool   `mangle:"user_vip"`
	Inventory int    `mangle:"fn_get_inventory"`
}

// PricingResponse represents the output
type PricingResponse struct {
	Price float64
}

func main() {
	// 1. Load Inventory (Simulate Redis)
	InventoryDB = make(map[string]int)
	wd, _ := os.Getwd()
	csvPath := filepath.Join(wd, "examples/dynamic_pricing/inventory.csv")
	file, err := os.Open(csvPath)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	reader := csv.NewReader(file)
	records, _ := reader.ReadAll()
	for _, record := range records {
		qty, _ := strconv.Atoi(record[1])
		InventoryDB[record[0]] = qty
	}
	fmt.Printf("Loaded %d products into InventoryDB (Redis Simulation)\n", len(InventoryDB))

	// 2. Initialize Client with Observability (Simplified DevEx)
	policyPath := filepath.Join(wd, "examples/dynamic_pricing/pricing.dl")

	client, err := sdk.NewClient(context.Background(),
		sdk.WithStdoutTracer(),
		sdk.WithPolicyPath(policyPath),
	)
	if err != nil {
		panic(err)
	}
	defer client.Shutdown(context.Background())

	fmt.Println("Client loaded with Tracing enabled.")

	// 3. Define Protected Action
	// The action itself just returns the standard price.
	// The policy will "Deny" it if a discount should be applied.
	// In this example, "Deny" = "Discount Applied".
	checkDiscount := sdk.ProtectFunc(client, "check_discount", func(ctx context.Context, req PricingRequest) (PricingResponse, error) {
		// Default behavior: Standard Price
		return PricingResponse{Price: 100.0}, nil
	})

	// 4. Stress Test
	iterations := 1000
	var totalDuration time.Duration

	fmt.Printf("Starting stress test with %d iterations...\n", iterations)

	ctx := context.Background()

	for i := 0; i < iterations; i++ {
		start := time.Now()

		// Simulate Request
		user := "user_vip_1"   // Assume this user is VIP
		product := "product_2" // Has 150 qty > 100

		// Call Native Tool (Go function)
		qty, found := get_inventory(product)
		if !found {
			qty = 0
		}

		// Prepare Input Struct
		req := PricingRequest{
			User:      user,
			Product:   product,
			IsVIP:     true,
			Inventory: qty,
		}

		// Execute Protected Action
		// We use sdk.Call helper for type safety
		// If Call returns ErrPolicyViolation, it means "deny" was derived -> Discount Applied
		_, err := sdk.Call[PricingResponse](ctx, checkDiscount, req)

		duration := time.Since(start)
		totalDuration += duration

		discountApplied := (err == core.ErrPolicyViolation)

		if i == 0 {
			if discountApplied {
				fmt.Printf("Iteration 0: Discount APPLIED for %s (Latency: %v)\n", product, duration)
			} else {
				fmt.Printf("Iteration 0: Discount NOT applied for %s (Latency: %v)\n", product, duration)
			}
		}
	}

	avgLatency := totalDuration / time.Duration(iterations)
	fmt.Printf("\n--- Results ---\n")
	fmt.Printf("Total Iterations: %d\n", iterations)
	fmt.Printf("Average Latency: %v\n", avgLatency)

	if avgLatency < 10*time.Millisecond {
		fmt.Println("✅ SUCCESS: Latency is under 10ms.")
	} else {
		fmt.Println("❌ FAILURE: Latency exceeded 10ms.")
	}
}
