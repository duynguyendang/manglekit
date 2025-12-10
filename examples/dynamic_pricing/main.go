package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/sdk"
)

// 1. Explicit Types
type PricingReq struct {
	User    string `mangle:"request_user"`
	Product string `mangle:"request_product"`
}

type PricingRes struct {
	Price float64 `json:"price"`
}

// 2. Pure Logic
func checkDiscount(ctx context.Context, req PricingReq) (PricingRes, error) {
	// Pure business logic: Standard Price
	return PricingRes{Price: 100.0}, nil
}

// InventoryDB simulates Redis
var InventoryDB map[string]int

func main() {
	// Load Inventory (Simulate Redis)
	InventoryDB = make(map[string]int)
	wd, _ := os.Getwd()
	csvPath := filepath.Join(wd, "examples/dynamic_pricing/inventory.csv")
	file, err := os.Open(csvPath)
	if err != nil {
		log.Fatalf("Failed to open inventory: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, _ := reader.ReadAll()
	for _, record := range records {
		qty, _ := strconv.Atoi(record[1])
		InventoryDB[record[0]] = qty
	}
	fmt.Printf("Loaded %d products.\n", len(InventoryDB))

	// Initialize Client (Use Facade)
	policyPath := filepath.Join(wd, "examples/dynamic_pricing/pricing.dl")

	client := manglekit.Must(manglekit.NewClient(context.Background(),
		sdk.WithStdoutTracer(),
		manglekit.WithBlueprintPath(policyPath),
	))
	defer client.Shutdown(context.Background())

	// 3. Generic Binding
	var CheckDiscount = manglekit.Define(client, "check_discount", checkDiscount)

	// 4. Stress Test
	iterations := 1000
	var totalDuration time.Duration

	ctx := context.Background()

	for i := 0; i < iterations; i++ {
		start := time.Now()

		user := "user_vip_1"
		product := "product_2"

		qty, found := InventoryDB[product]
		if !found {
			qty = 0
		}

		// 4. Context Injection
		// Injecting metadata facts directly using sdk.WithFact
		reqCtx := sdk.WithFact(ctx, "user_vip", "true")
		reqCtx = sdk.WithFact(reqCtx, "fn_get_inventory", strconv.Itoa(qty))

		req := PricingReq{
			User:    user,
			Product: product,
		}

		// 5. Type-Safe Execution
		_, err := CheckDiscount.Run(reqCtx, req)

		duration := time.Since(start)
		totalDuration += duration

		discountApplied := (err == core.ErrAlignment)

		if i == 0 {
			if discountApplied {
				fmt.Printf("Iteration 0: Discount APPLIED for %s\n", product)
			} else {
				fmt.Printf("Iteration 0: Discount NOT applied for %s\n", product)
			}
		}
	}

	avgLatency := totalDuration / time.Duration(iterations)
	fmt.Printf("\n--- Results ---\n")
	fmt.Printf("Total Iterations: %d\n", iterations)
	fmt.Printf("Average Latency: %v\n", avgLatency)
}
