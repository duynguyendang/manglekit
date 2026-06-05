package engine

import (
	"context"
	"errors"
	"testing"

	"github.com/duynguyendang/manglekit/core"
)

// TestTierAwareSolver_T0HaltBeatsT3Halt verifies that a T0 axiom halt
// outranks a T3 user-authored halt on the same envelope.
func TestTierAwareSolver_T0HaltBeatsT3Halt(t *testing.T) {
	engine, err := New()
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	rule := `
halt("Req", "T3 user violation", "T3").
halt("Req", "T0 axiom violation", "T0").
`
	if err := engine.runtime.AddPolicy(rule); err != nil {
		t.Fatalf("failed to load rule: %v", err)
	}

	input := core.NewEnvelope(map[string]string{"action": "test"})
	ctx := context.Background()
	err = engine.Assess(ctx, core.ActionMetadata{Name: "test"}, input)

	if !core.IsAlignmentError(err) {
		t.Fatal("expected alignment error")
	}

	var alignErr *core.AlignmentError
	if errors.As(err, &alignErr) {
		if alignErr.Message != "T0 axiom violation" {
			t.Errorf("expected T0 message 'T0 axiom violation', got %q", alignErr.Message)
		}
	}
}

// TestTierAwareSolver_T3HaltFiresWhenNoT0 verifies T3 halt works alone.
func TestTierAwareSolver_T3HaltFiresWhenNoT0(t *testing.T) {
	engine, err := New()
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	rule := `halt("Req", "T3 user violation", "T3").`
	if err := engine.runtime.AddPolicy(rule); err != nil {
		t.Fatalf("failed to load rule: %v", err)
	}

	input := core.NewEnvelope(map[string]string{"action": "test"})
	ctx := context.Background()
	err = engine.Assess(ctx, core.ActionMetadata{Name: "test"}, input)

	if !core.IsAlignmentError(err) {
		t.Fatal("expected alignment error from T3 halt")
	}

	var alignErr *core.AlignmentError
	if errors.As(err, &alignErr) {
		if alignErr.Message != "T3 user violation" {
			t.Errorf("expected 'T3 user violation', got %q", alignErr.Message)
		}
	}
}

// TestTierAwareSolver_BackwardCompat_Arity2 verifies that halt/2 (without tier)
// still works.
func TestTierAwareSolver_BackwardCompat_Arity2(t *testing.T) {
	engine, err := New()
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	rule := `halt("Req", "legacy halt").`
	if err := engine.runtime.AddPolicy(rule); err != nil {
		t.Fatalf("failed to load rule: %v", err)
	}

	input := core.NewEnvelope(map[string]string{"action": "test"})
	ctx := context.Background()
	err = engine.Assess(ctx, core.ActionMetadata{Name: "test"}, input)

	if !core.IsAlignmentError(err) {
		t.Fatal("expected alignment error from legacy halt/2")
	}

	var alignErr *core.AlignmentError
	if errors.As(err, &alignErr) {
		if alignErr.Message != "legacy halt" {
			t.Errorf("expected 'legacy halt', got %q", alignErr.Message)
		}
	}
}

// TestTierAwareSolver_Arity3BeatsArity2 verifies that a tier-attributed halt/3
// outranks a plain halt/2 (TierUnknown).
func TestTierAwareSolver_Arity3BeatsArity2(t *testing.T) {
	engine, err := New()
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	rule := `
halt("Req", "legacy no-tier halt").
halt("Req", "T1 governance halt", "T1").
`
	if err := engine.runtime.AddPolicy(rule); err != nil {
		t.Fatalf("failed to load rule: %v", err)
	}

	input := core.NewEnvelope(map[string]string{"action": "test"})
	ctx := context.Background()
	err = engine.Assess(ctx, core.ActionMetadata{Name: "test"}, input)

	if !core.IsAlignmentError(err) {
		t.Fatal("expected alignment error")
	}

	var alignErr *core.AlignmentError
	if errors.As(err, &alignErr) {
		if alignErr.Message != "T1 governance halt" {
			t.Errorf("expected T1 message, got %q", alignErr.Message)
		}
	}
}

// TestTierPriorityOrdering verifies the tier priority ordering.
func TestTierPriorityOrdering(t *testing.T) {
	tests := []struct {
		name     string
		tier     core.Tier
		expected int
	}{
		{"T0 axiom", core.TierT0_Axiom, 0},
		{"T1 governance", core.TierT1_Governance, 1},
		{"T2 playbook", core.TierT2_Playbook, 2},
		{"T3 user", core.TierT3_User, 3},
		{"unknown", core.TierUnknown, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tierPriority(tt.tier)
			if got != tt.expected {
				t.Errorf("tierPriority(%s) = %d, want %d", tt.tier, got, tt.expected)
			}
		})
	}

	if tierPriority(core.TierT0_Axiom) >= tierPriority(core.TierT3_User) {
		t.Error("T0 should outrank T3")
	}
}

// TestTierAwareSteering_RetryWithTier verifies retry/3 carries tier.
func TestTierAwareSteering_RetryWithTier(t *testing.T) {
	engine, err := New()
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	rule := `retry("Req", "fix alignment", "T1").`
	if err := engine.runtime.AddPolicy(rule); err != nil {
		t.Fatalf("failed to load rule: %v", err)
	}

	input := core.NewEnvelope(map[string]string{"action": "test"})
	ctx := context.Background()
	decision, meta, err := engine.EvaluateSteering(ctx, input)
	if err != nil {
		t.Fatalf("steering failed: %v", err)
	}

	if decision != core.DecisionRetry {
		t.Errorf("expected RETRY, got %s", decision)
	}
	if meta[core.KeyFeedback] != "fix alignment" {
		t.Errorf("expected feedback 'fix alignment', got %q", meta[core.KeyFeedback])
	}
}
