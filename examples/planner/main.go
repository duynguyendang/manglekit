package main

import (
	"context"
	"fmt"
	"os"
	"reflect"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/sdk"
)

func main() {
	ctx := context.Background()

	// 1. Create a temporary policy file defining the subgoals
	policyContent := `
subgoal("process_payment", "check_balance", 1).
subgoal("process_payment", "debit_account", 2).
subgoal("process_payment", "notify_user", 3).
`
	tmpFile, err := os.CreateTemp("", "policy_*.dl")
	if err != nil {
		panic(err)
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.WriteString(policyContent); err != nil {
		panic(err)
	}
	tmpFile.Close()

	// 2. Initialize Client with Policy
	client := manglekit.Must(manglekit.NewClient(ctx, manglekit.WithPolicyPath(tmpFile.Name())))

	// 3. Register Dummy Actions
	registerDummy(client, "check_balance")
	registerDummy(client, "debit_account")
	registerDummy(client, "notify_user")

	// 4. Plan
	fmt.Println("Generating Plan for 'process_payment'...")
	steps, err := client.Plan(ctx, "process_payment")
	if err != nil {
		panic(fmt.Errorf("planning failed: %w", err))
	}

	fmt.Printf("Plan Generated: %d steps\n", len(steps))
	for _, step := range steps {
		fmt.Printf(" - %d: %s\n", step.Order, step.ActionName)
	}

	// 5. Verify Plan
	expected := []sdk.PlanStep{
		{ActionName: "check_balance", Order: 1},
		{ActionName: "debit_account", Order: 2},
		{ActionName: "notify_user", Order: 3},
	}

	if !reflect.DeepEqual(steps, expected) {
		panic("Plan verification failed! Order or content mismatch.")
	}

	// 6. Execute Plan
	fmt.Println("\nExecuting Plan...")
	initialInput := core.NewEnvelope("start_payment")
	finalResult, err := client.ExecutePlan(ctx, steps, initialInput)
	if err != nil {
		panic(fmt.Errorf("execution failed: %w", err))
	}

	fmt.Printf("Execution Complete. Final Result: %v\n", finalResult.Payload)
}

func registerDummy(c *manglekit.Client, name string) {
	manglekit.Define(c, name, func(ctx context.Context, input string) (string, error) {
		fmt.Printf("  -> Executing action: %s (Input: %s)\n", name, input)
		return name + "_result", nil
	})
}
