package engine

import (
	"context"
	"testing"

	"github.com/duynguyendang/manglekit/core"
)

func TestPolicyEngine_AuthorizeWithSimpleDenyRule(t *testing.T) {
	// Create a new PolicyEngine
	engine := New()

	// Define a simple fact that violates policy
	// In Mangle, we can query for fact existence
	rule := `deny("Req").`
	if err := engine.runtime.LoadFromString(rule); err != nil {
		t.Fatalf("failed to load rule: %v", err)
	}

	// Test: Authorization should be denied
	input := core.NewEnvelope(map[string]string{"action": "test"})
	ctx := context.Background()
	err := engine.Authorize(ctx, core.ActionMetadata{Name: "test_action"}, input)

	if !core.IsAlignmentError(err) {
		t.Errorf("expected ErrAlignment, got %v", err)
	}
}

func TestPolicyEngine_AllowWhenNoRule(t *testing.T) {
	// Create a new PolicyEngine with empty runtime
	engine := New()

	// Don't load any rules - should allow by default
	input := core.NewEnvelope(map[string]string{"action": "test"})
	ctx := context.Background()
	err := engine.Authorize(ctx, core.ActionMetadata{Name: "test_action"}, input)

	if err != nil {
		t.Errorf("expected no error when no deny rule is defined, got %v", err)
	}
}

func TestMangleRuntime_ExecuteQuerySimple(t *testing.T) {
	runtime := NewMangleRuntime()

	// Load a simple fact
	rule := `allow("user").`
	if err := runtime.LoadFromString(rule); err != nil {
		t.Fatalf("failed to load rule: %v", err)
	}

	// Query for the fact
	allowed, err := runtime.ExecuteQuery(nil, `allow("user")`)
	if err != nil {
		t.Fatalf("query execution failed: %v", err)
	}

	if !allowed {
		t.Errorf("expected allow(user) to be true, got false")
	}
}

func TestMangleRuntime_ExecuteQueryWithFacts(t *testing.T) {
	runtime := NewMangleRuntime()

	// Load a rule that checks a fact
	rule := `user_is_admin("alice").`
	if err := runtime.LoadFromString(rule); err != nil {
		t.Fatalf("failed to load rule: %v", err)
	}

	// Query for a fact that exists
	exists, err := runtime.ExecuteQuery(nil, `user_is_admin("alice")`)
	if err != nil {
		t.Fatalf("query execution failed: %v", err)
	}

	if !exists {
		t.Errorf("expected user_is_admin(alice) to be true, got false")
	}

	// Query for a fact that doesn't exist
	notExists, err := runtime.ExecuteQuery(nil, `user_is_admin("bob")`)
	if err != nil {
		t.Fatalf("query execution failed: %v", err)
	}

	if notExists {
		t.Errorf("expected user_is_admin(bob) to be false, got true")
	}
}

func TestMangleRuntime_QueryWithSolutions_Number(t *testing.T) {
	runtime := NewMangleRuntime()

	// Load a rule that produces a number
	rule := `score("req", 100).`
	if err := runtime.LoadFromString(rule); err != nil {
		t.Fatalf("failed to load rule: %v", err)
	}

	// Query for solutions
	found := false
	err := runtime.QueryWithSolutions(nil, `score("req", X)`, func(sol map[string]any) error {
		val, ok := sol["X"]
		if !ok {
			t.Errorf("expected variable X in solution")
		}
		// Numbers are converted to string by constantToString
		if val != "100" {
			t.Errorf("expected value '100', got '%v'", val)
		}
		found = true
		return nil
	})
	if err != nil {
		t.Fatalf("query execution failed: %v", err)
	}
	if !found {
		t.Errorf("expected to find a solution")
	}
}
