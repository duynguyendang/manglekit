package main

import (
	"context"
	"fmt"
	"log"

	"github.com/duynguyendang/manglekit/v1/core"
	"github.com/duynguyendang/manglekit/v1/core/reflection"
	"github.com/duynguyendang/manglekit/policy/rulegenerator"
	"github.com/google/mangle/analysis"
	"github.com/google/mangle/ast"
	"github.com/google/mangle/engine"
	"github.com/google/mangle/factstore"
	"github.com/google/mangle/parse"
)

// Payment represents a sample transaction object.
type Payment struct {
	Amount int    `mangle:"amount"`
	Region string `mangle:"region"`
}

// mockLLM simulates an LLM's response for this example.
type mockLLM struct{}

func (m *mockLLM) Complete(ctx context.Context, req core.LLMRequest) (core.LLMResponse, error) {
	// This mock is now simplified. In a real scenario, it would adapt to the prompt.
	// For this example, we'll just return a rule that matches the custom "allow" head.
	expectedRule := `allow(Req) :- amount(Req, Amount), Amount < 100.`
	return core.LLMResponse{Text: expectedRule}, nil
}

func main() {
	ctx := context.Background()

	llm := &mockLLM{}
	// Demonstrate custom options. We'll change the rule head to "allow".
	opts := rulegenerator.GeneratorOptions{
		RuleHead: "allow",
		Examples: `
- User Policy: "Permit if amount < 100"
- Your Output:
allow(Req) :- amount(Req, Amount), Amount < 100.
`,
	}
	generator, err := rulegenerator.New(llm, opts)
	if err != nil {
		log.Fatalf("Failed to create generator: %v", err)
	}

	policyText := "Allow payment if amount is less than 100"
	samplePayment := Payment{}

	fmt.Printf("Natural Language Policy: \"%s\"\n", policyText)
	fmt.Println("--------------------------------------------------")

	generatedRule, err := generator.GenerateRule(ctx, samplePayment, policyText)
	if err != nil {
		log.Fatalf("Error generating rule: %v", err)
	}

	fmt.Printf("Generated Datalog Rule: %s\n", generatedRule)
	fmt.Println("--------------------------------------------------")

	// Test cases are updated to match the new "allow" policy.
	compliantPayment := Payment{Amount: 50, Region: "ANY"} // Should be allowed
	violatingPayment := Payment{Amount: 150, Region: "ANY"} // Should NOT be allowed

	if err := executeAndVerify(generatedRule, compliantPayment, "compliantPayment", true); err != nil {
		log.Fatalf("Verification failed for compliant payment: %v", err)
	}

	if err := executeAndVerify(generatedRule, violatingPayment, "violatingPayment", false); err != nil {
		log.Fatalf("Verification failed for violating payment: %v", err)
	}

	fmt.Println("✅ Policy Rule Generator demonstration successful!")
}

// executeAndVerify uses the mangle v0.3.0 Bottom-Up Materialization pattern.
func executeAndVerify(rule string, payment Payment, paymentID string, expectAllow bool) error {
	fmt.Printf("Verifying rule against '%s' (Amount: %d, Region: '%s')... Expecting allow=%v\n",
		paymentID, payment.Amount, payment.Region, expectAllow)

	// 1. Parse the rule and prepare the program.
	clause, err := parse.Clause(rule)
	if err != nil {
		return fmt.Errorf("failed to parse generated rule: %w", err)
	}
	program := []ast.Clause{clause}

	// 2. Create facts from the Go struct.
	facts, err := reflection.ToFacts(paymentID, payment)
	if err != nil {
		return fmt.Errorf("failed to convert payment to facts: %w", err)
	}

	// 3. Set up the fact store and add initial (EDB) facts.
	store := factstore.NewSimpleInMemoryStore()
	for _, fact := range facts {
		store.Add(fact)
	}

	// 4. Provide declarations for the EDB predicates to the analyzer.
	knownPredicates := make(map[ast.PredicateSym]ast.Decl)
	for _, fact := range facts {
		if _, ok := knownPredicates[fact.Predicate]; !ok {
			knownPredicates[fact.Predicate] = ast.NewSyntheticDeclFromSym(fact.Predicate)
		}
	}

	// 5. Analyze the program. The analyzer will discover the 'deny' predicate from the rule.
	programInfo, err := analysis.AnalyzeOneUnit(parse.SourceUnit{Clauses: program}, knownPredicates)
	if err != nil {
		return fmt.Errorf("failed to analyze program: %w", err)
	}

	// 6. Evaluate the program. This materializes all consequences, including 'deny' facts, into the store.
	if err := engine.EvalProgram(programInfo, store); err != nil {
		return fmt.Errorf("failed to evaluate program: %w", err)
	}

	// 7. Directly inspect the store for the result. No top-down query is needed.
	query, err := parse.Atom(fmt.Sprintf(`allow("%s")`, paymentID))
	if err != nil {
		return fmt.Errorf("failed to parse query atom: %w", err)
	}

	allowed := store.Contains(query)

	if allowed != expectAllow {
		return fmt.Errorf("unexpected outcome: got allow=%v, want allow=%v", allowed, expectAllow)
	}

	fmt.Printf("-> Correctly evaluated. Allowed: %v\n", allowed)
	return nil
}
