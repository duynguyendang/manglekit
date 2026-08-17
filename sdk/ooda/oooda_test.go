package ooda

import (
	"context"
	"testing"

	"github.com/duynguyendang/manglekit/core"
)

func TestRunOODA_Success(t *testing.T) {
	ctx := context.Background()

	registry := NewRegistry()
	registry.MustRegister("test_action", func(ctx context.Context, args map[string]interface{}) (string, error) {
		return "success", nil
	})

	brain := &mockBrain{
		decision: &core.Decision{
			Outcome: core.DecisionProceed,
			Action:  core.NewActionEnvelope("test_action", nil),
		},
	}

	frame := NewBuilder().
		WithInput("test tool execution").
		WithBrain(brain).
		WithRegistry(registry).
		Build()

	result, err := RunOODA(ctx, frame)
	if err != nil {
		t.Fatalf("RunOODA failed: %v", err)
	}

	if result.Status != VerifyStatusPassed {
		t.Errorf("expected status PASSED, got %s", result.Status)
	}

	if result.ActionResult != "success" {
		t.Errorf("expected action result 'success', got %v", result.ActionResult)
	}
}

func TestPinAxiom(t *testing.T) {
	frame := &CognitiveFrame{}
	atom := Atom{Predicate: "safety", Subject: "t0", Object: "no_deletes"}

	PinAxiom(frame, atom)

	if len(frame.AttentionSink) != 1 {
		t.Fatalf("expected 1 axiom, got %d", len(frame.AttentionSink))
	}
	if frame.AttentionSink[0].Weight != 1.0 {
		t.Errorf("expected weight 1.0, got %f", frame.AttentionSink[0].Weight)
	}
}

func TestShaveContext(t *testing.T) {
	frame := &CognitiveFrame{}
	for i := 0; i < 100; i++ {
		weight := float64(i%10) / 10.0
		AddContext(frame, Atom{
			Predicate: "fact",
			Subject:   "s",
			Object:    "o",
			Weight:    weight,
		})
	}

	before := len(frame.Context)
	ShaveContext(frame, 50)
	after := len(frame.Context)

	if after >= before {
		t.Errorf("expected shaving to reduce context: before=%d, after=%d", before, after)
	}
}

func TestVerifySchema_Success(t *testing.T) {
	ctx := context.Background()
	frame := &CognitiveFrame{
		Decision:     &core.Decision{Action: core.NewActionEnvelope("test", nil)},
		ActionResult: "success result",
	}

	err := VerifySchema(ctx, frame)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestVerifySchema_ErrorResult(t *testing.T) {
	ctx := context.Background()
	frame := &CognitiveFrame{
		Decision:     &core.Decision{Action: core.NewActionEnvelope("test", nil)},
		ActionResult: "error: something went wrong",
	}

	err := VerifySchema(ctx, frame)
	if err == nil {
		t.Error("expected error for error-like result")
	}
}

func TestShouldTerminate(t *testing.T) {
	tests := []struct {
		name       string
		audit      *AuditResult
		retryCount int
		maxRetry   int
		entropy    float64
		expected   string
	}{
		{
			name:       "T0 violation",
			audit:      &AuditResult{ViolationTier: Tier0Kernel},
			retryCount: 0,
			maxRetry:   3,
			entropy:    0.1,
			expected:   "t0_violation",
		},
		{
			name:       "max iterations",
			audit:      &AuditResult{ViolationTier: Tier1Admin},
			retryCount: 3,
			maxRetry:   3,
			entropy:    0.3,
			expected:   "max_iterations",
		},
		{
			name:       "chaos threshold",
			audit:      &AuditResult{ViolationTier: Tier2AI},
			retryCount: 1,
			maxRetry:   3,
			entropy:    0.95,
			expected:   "chaos_threshold",
		},
		{
			name:       "continue",
			audit:      &AuditResult{ViolationTier: Tier1Admin, Pass: false},
			retryCount: 1,
			maxRetry:   3,
			entropy:    0.3,
			expected:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldTerminate(tt.audit, tt.retryCount, tt.maxRetry, tt.entropy)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
