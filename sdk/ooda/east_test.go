package ooda

import (
	"context"
	"testing"

	"github.com/duynguyendang/manglekit/core"
)

// --- Mock implementations for EAST tests ---

type mockBrainWithVerify struct {
	decision *core.Decision
	verifyFn func(ctx context.Context, frame *CognitiveFrame) (*core.AuditTrail, error)
}

func (b *mockBrainWithVerify) Evaluate(ctx context.Context, frame *CognitiveFrame) (*core.Decision, error) {
	if b.decision != nil {
		return b.decision, nil
	}
	return &core.Decision{Outcome: core.DecisionProceed}, nil
}

func (b *mockBrainWithVerify) Verify(ctx context.Context, frame *CognitiveFrame) (*core.AuditTrail, error) {
	if b.verifyFn != nil {
		return b.verifyFn(ctx, frame)
	}
	return &core.AuditTrail{}, nil
}

func (b *mockBrainWithVerify) LoadPolicy(ctx context.Context, rules string) error {
	return nil
}

// --- EAST Tests ---

func TestMeasureSaliency(t *testing.T) {
	tests := []struct {
		input    string
		expected int // minimum signals
	}{
		{"normal request", 0},
		{"production database migration", 2}, // production + migration
		{"critical security deployment", 2},  // critical + security
		{"regular task", 0},
	}

	for _, tt := range tests {
		signals := MeasureSaliency(tt.input)
		if len(signals) < tt.expected {
			t.Errorf("MeasureSaliency(%q): expected >= %d signals, got %d", tt.input, tt.expected, len(signals))
		}
	}
}

func TestCalculateEntropy(t *testing.T) {
	tests := []struct {
		conflicts int
		total     int
		expected  float64
	}{
		{0, 10, 0.0},
		{5, 10, 0.5},
		{10, 10, 1.0},
		{0, 0, 0.0},
		{15, 10, 1.0}, // capped at 1.0
	}

	for _, tt := range tests {
		result := CalculateEntropy(tt.conflicts, tt.total)
		if result != tt.expected {
			t.Errorf("CalculateEntropy(%d, %d): expected %f, got %f", tt.conflicts, tt.total, tt.expected, result)
		}
	}
}

func TestCalculateActivity(t *testing.T) {
	atoms := []Atom{
		{Predicate: "rule", Subject: "r1", Object: "value1"},
		{Predicate: "rule", Subject: "r1", Object: "value1"}, // duplicate
		{Predicate: "rule", Subject: "r2", Object: "value2"},
	}

	activity := CalculateActivity(atoms)
	if activity["rule:r1"] != 2 {
		t.Errorf("expected rule:r1 count 2, got %d", activity["rule:r1"])
	}
	if activity["rule:r2"] != 1 {
		t.Errorf("expected rule:r2 count 1, got %d", activity["rule:r2"])
	}
}

func TestEASTState_Steer(t *testing.T) {
	tests := []struct {
		name     string
		state    EASTState
		frame    *CognitiveFrame
		expected ExecutionPath
	}{
		{
			name: "fast path: low entropy + T0 + known pattern",
			state: EASTState{
				Entropy:   0.1,
				TrustTier: Tier0Kernel,
			},
			frame: &CognitiveFrame{
				RawContext: map[string]any{"project_type": "modernization"},
			},
			expected: PathFast,
		},
		{
			name: "slow path: high entropy",
			state: EASTState{
				Entropy:   0.8,
				TrustTier: Tier0Kernel,
			},
			frame:    &CognitiveFrame{},
			expected: PathSlow,
		},
		{
			name: "slow path: low trust",
			state: EASTState{
				Entropy:   0.1,
				TrustTier: Tier3User,
			},
			frame:    &CognitiveFrame{},
			expected: PathSlow,
		},
		{
			name: "standard path",
			state: EASTState{
				Entropy:   0.4,
				TrustTier: Tier1Admin,
			},
			frame:    &CognitiveFrame{},
			expected: PathStandard,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.frame.EAST = tt.state
			result := tt.state.Steer(tt.frame)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestEASTState_CalculateMagnitude(t *testing.T) {
	state := EASTState{
		LogicSuccess:       0.5,
		EntropyCoefficient: 0.3,
	}
	magnitude := state.CalculateMagnitude()
	if magnitude <= 0 {
		t.Errorf("expected positive magnitude, got %f", magnitude)
	}
	if state.SteeringMagnitude != magnitude {
		t.Errorf("expected SteeringMagnitude to be set to %f, got %f", magnitude, state.SteeringMagnitude)
	}
}

func TestEASTState_Temperature(t *testing.T) {
	tests := []struct {
		magnitude float64
		expected  float64
	}{
		{0.1, 0.9}, // Creative
		{0.4, 0.7}, // Balanced
		{0.7, 0.4}, // Conservative
		{0.9, 0.1}, // Paradox
	}

	for _, tt := range tests {
		state := EASTState{SteeringMagnitude: tt.magnitude}
		result := state.Temperature()
		if result != tt.expected {
			t.Errorf("magnitude %f: expected temp %f, got %f", tt.magnitude, tt.expected, result)
		}
	}
}

func TestUpdateEntropy(t *testing.T) {
	tests := []struct {
		old        float64
		violations int
		total      int
		maxResult  float64
	}{
		{0.0, 0, 10, 0.0},
		{0.0, 5, 10, 0.5},
		{0.5, 5, 10, 1.0},
		{0.8, 5, 10, 1.0}, // capped
	}

	for _, tt := range tests {
		result := UpdateEntropy(tt.old, tt.violations, tt.total)
		if result > tt.maxResult {
			t.Errorf("UpdateEntropy(%f, %d, %d): expected <= %f, got %f", tt.old, tt.violations, tt.total, tt.maxResult, result)
		}
	}
}

// --- RunOODA Tests ---

func TestRunOODA_Success(t *testing.T) {
	ctx := context.Background()

	registry := NewRegistry()
	registry.MustRegister("test_action", func(ctx context.Context, args map[string]interface{}) (string, error) {
		return "success", nil
	})

	brain := &mockBrainWithVerify{
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

// --- RunOODAEAST Tests ---

func TestRunOODAEAST_Success(t *testing.T) {
	ctx := context.Background()

	registry := NewRegistry()
	registry.MustRegister("generate", func(ctx context.Context, args map[string]interface{}) (string, error) {
		return "generated content", nil
	})

	brain := &mockBrainWithVerify{
		decision: &core.Decision{
			Outcome: core.DecisionProceed,
			Action:  core.NewActionEnvelope("generate", nil),
		},
		verifyFn: func(ctx context.Context, frame *CognitiveFrame) (*core.AuditTrail, error) {
			return &core.AuditTrail{}, nil // No violations
		},
	}

	frame := NewBuilder().
		WithInput("generate document for production system").
		WithBrain(brain).
		WithRegistry(registry).
		Build()

	result, err := RunOODAEAST(ctx, frame)
	if err != nil {
		t.Fatalf("RunOODAEAST failed: %v", err)
	}

	if result.Status != VerifyStatusPassed {
		t.Errorf("expected status PASSED, got %s", result.Status)
	}

	// Verify EAST state was calculated
	if result.EAST.Saliency == nil {
		t.Error("expected saliency to be measured")
	}
}

func TestRunOODAEAST_FastPath(t *testing.T) {
	ctx := context.Background()

	registry := NewRegistry()
	registry.MustRegister("fast_action", func(ctx context.Context, args map[string]interface{}) (string, error) {
		return "fast result", nil
	})

	brain := &mockBrainWithVerify{
		decision: &core.Decision{
			Outcome: core.DecisionProceed,
			Action:  core.NewActionEnvelope("fast_action", nil),
		},
		verifyFn: func(ctx context.Context, frame *CognitiveFrame) (*core.AuditTrail, error) {
			return &core.AuditTrail{}, nil
		},
	}

	frame := NewBuilder().
		WithInput("standard request").
		WithBrain(brain).
		WithRegistry(registry).
		Build()

	// Set up for fast-path: low entropy + T0 + known pattern
	frame.EAST.Entropy = 0.1
	frame.EAST.TrustTier = Tier0Kernel
	frame.EAST.Saliency = []string{"known_pattern"}
	frame.AttentionSink = []Atom{
		{Predicate: "axiom", Subject: "safety", Object: "halt_on_delete", Weight: 1.0},
	}
	frame.RawContext = map[string]any{"project_type": "modernization"}

	result, err := RunOODAEAST(ctx, frame)
	if err != nil {
		t.Fatalf("RunOODAEAST failed: %v", err)
	}

	if result.Status != VerifyStatusPassed {
		t.Errorf("expected status PASSED, got %s", result.Status)
	}
}

// --- Memory Tests ---

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
	// Add atoms with varying weights
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
	ShaveContext(frame, 50) // Very small budget
	after := len(frame.Context)

	if after >= before {
		t.Errorf("expected shaving to reduce context: before=%d, after=%d", before, after)
	}
}

// --- Verify Tests ---

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

func TestEASTState_ShouldInjectParadox_DefaultThreshold(t *testing.T) {
	high := EASTState{SteeringMagnitude: 0.81}
	if !high.ShouldInjectParadox() {
		t.Error("expected paradox injection at magnitude 0.81 (default 0.8)")
	}
	at := EASTState{SteeringMagnitude: 0.8}
	if at.ShouldInjectParadox() {
		t.Error("expected no paradox injection exactly at threshold 0.8")
	}
	low := EASTState{SteeringMagnitude: 0.5}
	if low.ShouldInjectParadox() {
		t.Error("expected no paradox injection at magnitude 0.5")
	}
}

func TestEASTState_ShouldInjectParadox_CustomThreshold(t *testing.T) {
	customLow := EASTState{SteeringMagnitude: 0.6, ParadoxThreshold: 0.5}
	if !customLow.ShouldInjectParadox() {
		t.Error("expected paradox injection with custom threshold 0.5 at magnitude 0.6")
	}
	customHigh := EASTState{SteeringMagnitude: 0.6, ParadoxThreshold: 0.9}
	if customHigh.ShouldInjectParadox() {
		t.Error("expected no paradox injection with custom threshold 0.9 at magnitude 0.6")
	}
}
