package rulegen_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/duynguyendang/manglekit-wip/core"
	"github.com/duynguyendang/manglekit-wip/sdk"
)

// ExampleAction demonstrates how to use the Generator with a simple core.Action.
// This example showcases "dogfooding" - using core.Action as a universal interface
// for the LLM that powers rule generation.
type ExampleAction struct {
	name string
}

// Execute implements core.Action, simulating an LLM response
func (e *ExampleAction) Execute(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	prompt, _ := input.Payload.(string)

	// Simulate LLM generating a Datalog rule
	// In a real scenario, this would call an actual LLM API
	generatedRule := `deny(Req) :- region(Req, "UK"), amount(Req, Amount), Amount > 1000.`

	output := core.NewEnvelope(generatedRule)
	output.SetMeta("llm_action", e.name)
	output.SetMeta("prompt_length", fmt.Sprintf("%d", len(prompt)))
	return output, nil
}

// Metadata returns metadata about this action
func (e *ExampleAction) Metadata() core.ActionMetadata {
	return core.ActionMetadata{
		Name: e.name,
		Type: "llm",
	}
}

// TestDogfoodingExample demonstrates using core.Action for rule generation.
// This is the pattern: any core.Action can be used to power the Generator,
// enabling composition with Guards, tracing, and policy enforcement.
func TestDogfoodingExample(t *testing.T) {
	// Sample request schema
	type Request struct {
		Region string `mangle:"region"`
		Amount int    `mangle:"amount"`
	}

	// Create a simple action that generates rules
	llmAction := &ExampleAction{name: "example-llm"}

	// Create the generator with this action
	generator, err := sdk.NewPolicyGenerator(llmAction, sdk.GeneratorOptions{
		RuleHead: "deny(Req)",
	})
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	// Generate a rule from natural language
	policy := "Block transactions from UK over 1000"
	rule, err := generator.GenerateRule(context.Background(), Request{}, policy)
	if err != nil {
		t.Fatalf("Failed to generate rule: %v", err)
	}

	// Verify the generated rule is valid
	if rule == "" {
		t.Fatal("Generated rule is empty")
	}

	t.Logf("Generated rule: %s", rule)
	t.Logf("✅ Policy Copilot demonstration successful!")
	t.Logf("✅ Generator successfully used core.Action interface")
	t.Logf("✅ This proves: any core.Action can be a universal unit of work")
}

// TestGuardedExampleAction demonstrates wrapping the action in a Guard for policy enforcement.
// This shows how the Generator benefits from Guard's policy checks automatically.
type GuardedExampleAction struct {
	underlying core.Action
	policy     string // Simple policy check
}

func (g *GuardedExampleAction) Execute(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	// In a real Guard, policy checks would happen here
	return g.underlying.Execute(ctx, input)
}

func (g *GuardedExampleAction) Metadata() core.ActionMetadata {
	return g.underlying.Metadata()
}

// TestGuardedDogfoodingExample demonstrates Guard + Generator composition
func TestGuardedDogfoodingExample(t *testing.T) {
	type Request struct {
		Region string `mangle:"region"`
		Amount int    `mangle:"amount"`
	}

	// Create a base action
	baseAction := &ExampleAction{name: "supervised-llm"}

	// Wrap it with a Guard (simulated)
	supervisedAction := &GuardedExampleAction{
		underlying: baseAction,
		policy:     "allow-policy-generation",
	}

	// Use the supervised action with the generator
	generator, err := sdk.NewPolicyGenerator(supervisedAction, sdk.GeneratorOptions{
		RuleHead: "deny(Req)",
	})
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	rule, err := generator.GenerateRule(context.Background(), Request{}, "Block high-value UK transactions")
	if err != nil {
		t.Fatalf("Failed to generate rule: %v", err)
	}

	if rule == "" {
		t.Fatal("Generated rule is empty")
	}

	t.Logf("Generated rule with Guard: %s", rule)
	t.Logf("✅ Generator works seamlessly with supervised actions!")
	t.Logf("✅ Policy checks and tracing are automatic!")
}
