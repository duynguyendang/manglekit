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
	"fmt"
	"os"
	"strings"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/logger"
	"github.com/duynguyendang/manglekit/policy/rulegenerator"
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
// Rule Generator Action (Wraps rulegenerator in a core.Action)
// =============================================================================

// RuleGeneratorAction wraps the rulegenerator package as a core.Action.
// This allows LLM-based rule generation to be governed by the Manglekit guard.
type RuleGeneratorAction struct {
	name      string
	generator *rulegenerator.Generator
	schema    any // The sample struct for schema extraction
}

// NewRuleGeneratorAction creates a new action for generating Datalog rules.
func NewRuleGeneratorAction(name string, generator *rulegenerator.Generator, schema any) *RuleGeneratorAction {
	return &RuleGeneratorAction{
		name:      name,
		generator: generator,
		schema:    schema,
	}
}

// Execute takes a natural language policy (string payload) and returns the generated Datalog rule.
func (r *RuleGeneratorAction) Execute(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	// Retrieve logger from context (auto-injected by guard)
	log := core.LoggerFromContext(ctx)
	log.Debug("executing rule generator action", "action", r.name)

	policyText, ok := input.Payload.(string)
	if !ok {
		return core.Envelope{}, fmt.Errorf("invalid input type, expected string but got %T", input.Payload)
	}

	// Generate the Datalog rule from natural language
	generatedRule, err := r.generator.GenerateRule(ctx, r.schema, policyText)
	if err != nil {
		log.Error("rule generation failed", "error", err)
		return core.Envelope{}, fmt.Errorf("rule generation failed: %w", err)
	}

	log.Info("rule generated successfully", "rule", generatedRule)

	output := manglekit.NewEnvelope(generatedRule)
	output.SetMeta("action_name", r.name)
	output.SetMeta("policy_text", policyText)

	return output, nil
}

// Metadata returns the metadata for this rule generator action.
func (r *RuleGeneratorAction) Metadata() core.ActionMetadata {
	return core.ActionMetadata{
		Name: r.name,
		Type: "rule-generator",
	}
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
	// 1. Initialize Logger (using standard logger for demo)
	// ---------------------------------------------------------------------------
	log := logger.NewStdLogger()

	// ---------------------------------------------------------------------------
	// 2. Initialize the Manglekit Client with Logger
	// ---------------------------------------------------------------------------
	client, err := manglekit.NewClient(ctx, "", manglekit.WithLogger(log))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize manglekit client: %v\n", err)
		os.Exit(1)
	}
	log.Info("Manglekit client initialized", "logging", true)

	// ---------------------------------------------------------------------------
	// 3. Create the Rule Generator with Mock LLM
	// ---------------------------------------------------------------------------
	llm := &MockLLM{logger: log}
	generator, err := rulegenerator.New(llm, rulegenerator.GeneratorOptions{
		RuleHead: "deny(Req)", // The target predicate for our policy
	})
	if err != nil {
		log.Error("Failed to create generator", "error", err)
		os.Exit(1)
	}

	// Define a sample struct to teach the LLM our schema
	sampleTransaction := Transaction{}

	// ---------------------------------------------------------------------------
	// 4. Wrap the Rule Generator in a core.Action
	// ---------------------------------------------------------------------------
	rawAction := NewRuleGeneratorAction("policy-copilot", generator, sampleTransaction)
	log.Info("RuleGeneratorAction created")

	// ---------------------------------------------------------------------------
	// 5. Protect the Action with Governance Guard
	// ---------------------------------------------------------------------------
	safeRuleGenerator := client.Protect(rawAction)
	log.Info("Action protected with governance guard")
	fmt.Println()

	// ---------------------------------------------------------------------------
	// 6. Execute Rule Generation (Governed)
	// ---------------------------------------------------------------------------
	policyText := "Block transactions over 1000 in the UK"

	fmt.Printf("📝 Natural Language Policy:\n   \"%s\"\n\n", policyText)
	fmt.Println("🔄 Generating Datalog rule from natural language (governed)...")
	fmt.Println()

	// Create input envelope with the policy text
	input := manglekit.NewEnvelope(policyText)

	// Execute the governed action
	output, err := safeRuleGenerator.Execute(ctx, input)
	if err != nil {
		log.Error("Error generating rule", "error", err)
		os.Exit(1)
	}

	generatedRule, ok := output.Payload.(string)
	if !ok {
		log.Error("Invalid output type", "type", fmt.Sprintf("%T", output.Payload))
		os.Exit(1)
	}

	fmt.Printf("✅ Generated Datalog Rule:\n   %s\n\n", generatedRule)
	fmt.Printf("   Action: %s\n", output.GetMeta("action_name"))
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// ---------------------------------------------------------------------------
	// 7. Create Evaluator and Test the Generated Rule
	// ---------------------------------------------------------------------------
	evaluator, err := rulegenerator.NewEvaluator(generatedRule)
	if err != nil {
		log.Error("Error creating evaluator", "error", err)
		os.Exit(1)
	}

	fmt.Println("🧪 Testing Generated Rule Against Sample Transactions:")
	fmt.Println()

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
		// Use the encapsulated evaluator
		result, err := evaluator.Evaluate(tc.name, tc.transaction)
		if err != nil {
			log.Error("Error evaluating test case", "name", tc.name, "error", err)
			allPassed = false
			continue
		}

		if result.Matched == tc.expectDeny {
			fmt.Printf("   ✅ PASS: %s\n", tc.name)
			fmt.Printf("      Amount: %d, Region: %s → Denied: %v (expected: %v)\n\n",
				tc.transaction.Amount, tc.transaction.Region, result.Matched, tc.expectDeny)
		} else {
			fmt.Printf("   ❌ FAIL: %s\n", tc.name)
			fmt.Printf("      Amount: %d, Region: %s → Denied: %v (expected: %v)\n\n",
				tc.transaction.Amount, tc.transaction.Region, result.Matched, tc.expectDeny)
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
