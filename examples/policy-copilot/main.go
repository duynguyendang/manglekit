// Example: Policy Copilot - Natural Language to Datalog Compiler
//
// This example demonstrates the Manglekit governance pattern applied to
// policy generation:
//  1. Define a Go struct representing your data model (Transaction)
//  2. Use natural language to express a policy ("Block transactions over 1000 in the UK")
//  3. Wrap the LLM-based rule generation in a governed Action
//  4. Generate executable Datalog rules from the natural language (with governance)
//  5. Verify the generated rules work correctly against test data
//
// The Policy Copilot leverages:
//   - Manglekit's governance guard for LLM calls
//   - Reflection to extract schema from Go structs
//   - LLM to translate natural language to Datalog
//   - Internal Mangle engine for syntax verification and execution
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/duynguyendang/manglekit"
	funcAdapter "github.com/duynguyendang/manglekit/adapters/func"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/sdk"
)

// =============================================================================
// Domain Model
// =============================================================================

// Transaction represents a financial transaction that policies will be applied to.
// The `mangle` struct tags define how fields map to Datalog predicates.
type Transaction struct {
	Amount int    `mangle:"amount"` // amount(EntityID, IntValue)
	Region string `mangle:"region"` // region(EntityID, StringValue)
}

// =============================================================================
// Mock LLM (for demonstration - replace with real LLM in production)
// =============================================================================

// MockLLM simulates an LLM response for this example.
// In production, you would use a real LLM client (OpenAI, Google Gemini, etc.)
// It implements ai.TextGenerator interface.
type MockLLM struct {
	logger core.Logger
}

func (m *MockLLM) Complete(ctx context.Context, prompt string) (string, error) {
	m.logger.Info("MockLLM received prompt for rule generation")

	// Simulate understanding the prompt and generating appropriate Datalog
	// In reality, this would be an API call to GPT-4, Gemini, etc.
	if strings.Contains(prompt, "over 1000") && strings.Contains(prompt, "UK") {
		return `deny(Req) :- amount(Req, Amount), region(Req, "UK"), Amount > 1000.`, nil
	}
	if strings.Contains(prompt, "less than 100") {
		return `allow(Req) :- amount(Req, Amount), Amount < 100.`, nil
	}

	// Default response
	return `deny(Req) :- amount(Req, X), X > 1000.`, nil
}

// =============================================================================
// Main Demo
// =============================================================================

func main() {
	ctx := context.Background()

	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║         Policy Copilot - NL to Datalog Compiler Demo          ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// ---------------------------------------------------------------------------
	// 2. Initialize the Manglekit Client (uses default Zap logger)
	// ---------------------------------------------------------------------------
	client := manglekit.Must(manglekit.NewClient(ctx))
	log := client.Logger()
	log.Info("Manglekit client initialized", "logging", true)

	// ---------------------------------------------------------------------------
	// 3. Create the Rule Generator with Mock LLM
	// ---------------------------------------------------------------------------
	llm := &MockLLM{logger: log}
	llmAction := funcAdapter.New("mockLLM", llm.Complete)
	generator, err := sdk.NewPolicyGenerator(llmAction, sdk.GeneratorOptions{
		RuleHead: "deny(Req)", // The target predicate for our policy
	})
	if err != nil {
		log.Error("Failed to create generator", "error", err)
		os.Exit(1)
	}

	// Define a sample struct to teach the LLM our schema
	sampleTransaction := Transaction{}

	// ---------------------------------------------------------------------------
	// 4. Define Logic as a Closure and Protect it
	// ---------------------------------------------------------------------------
	genLogic := func(ctx context.Context, policy string) (string, error) {
		return generator.GenerateRule(ctx, sampleTransaction, policy)
	}

	action := manglekit.Define(client, "policy-copilot", genLogic)
	log.Info("Rule generation logic protected with governance guard")
	fmt.Println()

	// ---------------------------------------------------------------------------
	// 5. Execute Rule Generation (Governed)
	// ---------------------------------------------------------------------------
	policyText := "Block transactions over 1000 in the UK"

	fmt.Printf("📝 Natural Language Policy:\n   \"%s\"\n\n", policyText)
	fmt.Println("🔄 Generating Datalog rule from natural language (governed)...")
	fmt.Println()

	// Use the generic Call helper for a clean, type-safe execution
	generatedRule, err := action.Run(ctx, policyText)
	if err != nil {
		log.Error("Error generating rule", "error", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Generated Datalog Rule:\n   %s\n\n", generatedRule)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// ---------------------------------------------------------------------------
	// 7. Verify the Generated Rule (Using Manglekit Facade)
	// ---------------------------------------------------------------------------
	fmt.Println("🧪 Testing Generated Rule Against Sample Transactions:")
	fmt.Println()

	// Write the rule to a temporary file
	tmpFile, err := os.CreateTemp("", "policy_*.dl")
	if err != nil {
		log.Error("Failed to create temp file", "error", err)
		os.Exit(1)
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.Write([]byte(generatedRule)); err != nil {
		log.Error("Failed to write rule to file", "error", err)
		os.Exit(1)
	}
	tmpFile.Close()

	// Create a new client that uses this generated policy
	// We use "open" mode so that we only block if the generated policy explicitly denies.
	// Default is "closed" which blocks everything not explicitly allowed, which would fail valid tests.
	verifierClient := manglekit.Must(manglekit.NewClient(
		ctx,
		manglekit.WithPolicyPath(tmpFile.Name()),
		manglekit.WithFailMode("open"),
	))

	// Define a dummy action to test the policy against
	// We return "allowed" by default. If policy denies, we get an error.
	verifyAction := manglekit.Define(verifierClient, "verify_tx", func(ctx context.Context, t Transaction) (string, error) {
		return "allowed", nil
	})

	testCases := []struct {
		name        string
		transaction Transaction
		expectDeny  bool
	}{
		{
			name:        "UK transaction over 1000 (should be denied)",
			transaction: Transaction{Amount: 1500, Region: "UK"},
			expectDeny:  true,
		},
		{
			name:        "UK transaction under 1000 (should be allowed)",
			transaction: Transaction{Amount: 500, Region: "UK"},
			expectDeny:  false,
		},
		{
			name:        "US transaction over 1000 (should be allowed)",
			transaction: Transaction{Amount: 2000, Region: "US"},
			expectDeny:  false,
		},
		{
			name:        "FR transaction exactly 1000 (should be allowed)",
			transaction: Transaction{Amount: 1000, Region: "FR"},
			expectDeny:  false,
		},
	}

	allPassed := true
	for _, tc := range testCases {
		_, err := verifyAction.Run(ctx, tc.transaction)

		isDenied := false
		var pve *core.PolicyViolationError
		if errors.As(err, &pve) {
			isDenied = true
		} else if err != nil {
			log.Error("Unexpected error evaluating test case", "name", tc.name, "error", err)
			allPassed = false
			continue
		}

		if isDenied == tc.expectDeny {
			fmt.Printf("   ✅ PASS: %s\n", tc.name)
			fmt.Printf("      Amount: %d, Region: %s → Denied: %v (expected: %v)\n\n",
				tc.transaction.Amount, tc.transaction.Region, isDenied, tc.expectDeny)
		} else {
			fmt.Printf("   ❌ FAIL: %s\n", tc.name)
			fmt.Printf("      Amount: %d, Region: %s → Denied: %v (expected: %v)\n\n",
				tc.transaction.Amount, tc.transaction.Region, isDenied, tc.expectDeny)
			allPassed = false
		}
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	if allPassed {
		fmt.Println("🎉 All tests passed! Policy Copilot demonstration successful.")
	} else {
		fmt.Println("⚠️  Some tests failed. Check the output above.")
	}
}
