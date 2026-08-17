package east

import (
	"context"
	"testing"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/sdk/ooda"
	"github.com/duynguyendang/manglekit/sdk/ports"
)

type mockBrainWithVerify struct {
	decision *core.Decision
	verifyFn func(ctx context.Context, frame *ooda.CognitiveFrame) (*core.AuditTrail, error)
}

func (b *mockBrainWithVerify) Evaluate(ctx context.Context, frame *ooda.CognitiveFrame) (*core.Decision, error) {
	if b.decision != nil {
		return b.decision, nil
	}
	return &core.Decision{Outcome: core.DecisionProceed}, nil
}

func (b *mockBrainWithVerify) Verify(ctx context.Context, frame *ooda.CognitiveFrame) (*core.AuditTrail, error) {
	if b.verifyFn != nil {
		return b.verifyFn(ctx, frame)
	}
	return &core.AuditTrail{}, nil
}

func (b *mockBrainWithVerify) LoadPolicy(ctx context.Context, rules string) error {
	return nil
}

func TestMeasureSaliency(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"normal request", 0},
		{"production database migration", 2},
		{"critical security deployment", 2},
		{"regular task", 0},
	}
	for _, tt := range tests {
		if n := len(MeasureSaliency(tt.input)); n < tt.expected {
			t.Errorf("MeasureSaliency(%q): expected >= %d, got %d", tt.input, tt.expected, n)
		}
	}
}

func TestCalculateEntropy(t *testing.T) {
	cases := []struct {
		c, total int
		want     float64
	}{{0, 10, 0}, {5, 10, 0.5}, {10, 10, 1}, {0, 0, 0}, {15, 10, 1}}
	for _, tt := range cases {
		if got := CalculateEntropy(tt.c, tt.total); got != tt.want {
			t.Errorf("CalculateEntropy(%d,%d)=%v want %v", tt.c, tt.total, got, tt.want)
		}
	}
}

func TestCalculateActivity(t *testing.T) {
	atoms := []ooda.Atom{
		{Predicate: "rule", Subject: "r1", Object: "value1"},
		{Predicate: "rule", Subject: "r1", Object: "value1"},
		{Predicate: "rule", Subject: "r2", Object: "value2"},
	}
	activity := CalculateActivity(atoms)
	if activity["rule:r1"] != 2 || activity["rule:r2"] != 1 {
		t.Errorf("unexpected activity map: %v", activity)
	}
}

func TestSteer(t *testing.T) {
	cases := []struct {
		name  string
		state ooda.EASTState
		frame *ooda.CognitiveFrame
		want  ooda.ExecutionPath
	}{
		{"fast", ooda.EASTState{Entropy: 0.1, TrustTier: ooda.Tier0Kernel},
			&ooda.CognitiveFrame{RawContext: map[string]any{"project_type": "modernization"}}, ooda.PathFast},
		{"slow-high-entropy", ooda.EASTState{Entropy: 0.8, TrustTier: ooda.Tier0Kernel}, &ooda.CognitiveFrame{}, ooda.PathSlow},
		{"slow-low-trust", ooda.EASTState{Entropy: 0.1, TrustTier: ooda.Tier3User}, &ooda.CognitiveFrame{}, ooda.PathSlow},
		{"standard", ooda.EASTState{Entropy: 0.4, TrustTier: ooda.Tier1Admin}, &ooda.CognitiveFrame{}, ooda.PathStandard},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			tt.frame.EAST = tt.state
			if got := Steer(&tt.state, tt.frame); got != tt.want {
				t.Errorf("Steer=%v want %v", got, tt.want)
			}
		})
	}
}

func TestCalculateMagnitude(t *testing.T) {
	state := &ooda.EASTState{LogicSuccess: 0.5, EntropyCoefficient: 0.3}
	m := CalculateMagnitude(state)
	if m <= 0 || state.SteeringMagnitude != m {
		t.Errorf("unexpected magnitude %f / field %f", m, state.SteeringMagnitude)
	}
}

func TestTemperature(t *testing.T) {
	cases := []struct {
		m    float64
		want float64
	}{{0.1, 0.9}, {0.4, 0.7}, {0.7, 0.4}, {0.9, 0.1}}
	for _, tt := range cases {
		if got := Temperature(&ooda.EASTState{SteeringMagnitude: tt.m}); got != tt.want {
			t.Errorf("Temperature(%v)=%v want %v", tt.m, got, tt.want)
		}
	}
}

func TestUpdateEntropy(t *testing.T) {
	cases := []struct {
		old      float64
		v, total int
	}{{0, 0, 10}, {0, 5, 10}, {0.5, 5, 10}, {0.8, 5, 10}}
	for _, tt := range cases {
		if got := UpdateEntropy(tt.old, tt.v, tt.total); got > 1.0 {
			t.Errorf("UpdateEntropy(%v,%d,%d)=%v capped above 1", tt.old, tt.v, tt.total, got)
		}
	}
}

func TestShouldInjectParadox(t *testing.T) {
	if !ShouldInjectParadox(&ooda.EASTState{SteeringMagnitude: 0.81}) {
		t.Error("expected paradox at 0.81 (default 0.8)")
	}
	if ShouldInjectParadox(&ooda.EASTState{SteeringMagnitude: 0.8}) {
		t.Error("expected no paradox exactly at 0.8")
	}
	if ShouldInjectParadox(&ooda.EASTState{SteeringMagnitude: 0.5}) {
		t.Error("expected no paradox at 0.5")
	}
	if !ShouldInjectParadox(&ooda.EASTState{SteeringMagnitude: 0.6, ParadoxThreshold: 0.5}) {
		t.Error("expected paradox with custom threshold 0.5")
	}
	if ShouldInjectParadox(&ooda.EASTState{SteeringMagnitude: 0.6, ParadoxThreshold: 0.9}) {
		t.Error("expected no paradox with custom threshold 0.9")
	}
}

func newEastFrame(t *testing.T, input string, brain *mockBrainWithVerify) *ooda.CognitiveFrame {
	t.Helper()
	registry := ooda.NewRegistry()
	registry.MustRegister("generate", func(ctx context.Context, args map[string]interface{}) (string, error) {
		return "generated content", nil
	})
	return ooda.NewBuilder().WithInput(input).WithBrain(brain).WithRegistry(registry).Build()
}

func TestRunOODAEAST_Success(t *testing.T) {
	ctx := context.Background()
	brain := &mockBrainWithVerify{
		decision: &core.Decision{Outcome: core.DecisionProceed, Action: core.NewActionEnvelope("generate", nil)},
		verifyFn: func(ctx context.Context, frame *ooda.CognitiveFrame) (*core.AuditTrail, error) {
			return &core.AuditTrail{}, nil
		},
	}
	frame := newEastFrame(t, "generate document for production system", brain)

	result, err := RunOODAEAST(ctx, frame)
	if err != nil {
		t.Fatalf("RunOODAEAST failed: %v", err)
	}
	if result.Status != ooda.VerifyStatusPassed {
		t.Errorf("expected PASSED, got %s", result.Status)
	}
	if result.EAST.Saliency == nil {
		t.Error("expected saliency to be measured")
	}
}

func TestRunOODAEAST_FastPath(t *testing.T) {
	ctx := context.Background()
	brain := &mockBrainWithVerify{
		decision: &core.Decision{Outcome: core.DecisionProceed, Action: core.NewActionEnvelope("generate", nil)},
		verifyFn: func(ctx context.Context, frame *ooda.CognitiveFrame) (*core.AuditTrail, error) {
			return &core.AuditTrail{}, nil
		},
	}
	frame := newEastFrame(t, "standard request", brain)
	frame.EAST = ooda.EASTState{Entropy: 0.1, TrustTier: ooda.Tier0Kernel, Saliency: []string{"known_pattern"}}
	frame.AttentionSink = []ooda.Atom{{Predicate: "axiom", Subject: "safety", Object: "halt_on_delete", Weight: 1.0}}
	frame.RawContext = map[string]any{"project_type": "modernization"}

	result, err := RunOODAEAST(ctx, frame)
	if err != nil {
		t.Fatalf("RunOODAEAST failed: %v", err)
	}
	if result.Status != ooda.VerifyStatusPassed {
		t.Errorf("expected PASSED, got %s", result.Status)
	}
}

type mockReasoningPort struct {
	rules map[string]bool
}

var _ ports.ReasoningPort = (*mockReasoningPort)(nil)

func (m *mockReasoningPort) VerifyWithDatalog(_ context.Context, query string) ([]map[string]string, error) {
	if m.rules[query] {
		return []map[string]string{{"path": "true"}}, nil
	}
	return nil, nil
}

func TestCalculateEAST(t *testing.T) {
	mag := CalculateEAST(1.0, 2.0)
	if mag != 0.5 {
		t.Errorf("expected P = exp(0)/2 = 0.5, got %f", mag)
	}
	if clamped := CalculateEAST(2.0, 0.0); clamped != 100.0 {
		t.Errorf("expected clamped P = exp(0)/0.01 = 100, got %f", clamped)
	}
}

func TestGetSteeringPrompts(t *testing.T) {
	h, b := GetSteeringPrompts(0.9)
	if h == "" || b == "" {
		t.Error("expected non-empty prompt keys for high magnitude")
	}
	_, _ = GetSteeringPrompts(0.0)
}

func TestSteerKB(t *testing.T) {
	ctx := context.Background()

	t.Run("fast path via datalog", func(t *testing.T) {
		frame := &ooda.CognitiveFrame{}
		reasoner := &mockReasoningPort{rules: map[string]bool{}}
		if p := SteerKB(ctx, &ooda.EASTState{}, frame, reasoner); p != ooda.PathStandard {
			t.Errorf("expected standard path, got %d", p)
		}
	})

	t.Run("nil reasoner falls back to Steer", func(t *testing.T) {
		frame := &ooda.CognitiveFrame{}
		state := &ooda.EASTState{Entropy: 0.1, TrustTier: ooda.Tier0Kernel}
		frame.EAST.Saliency = []string{"known_pattern"}
		if p := SteerKB(ctx, state, frame, nil); p != ooda.PathFast {
			t.Errorf("expected fast path, got %d", p)
		}
	})
}

func TestIsSaliencyHighAndKnown(t *testing.T) {
	if !IsSaliencyHigh([]string{"high_priority"}) {
		t.Error("expected high_priority flagged")
	}
	if IsSaliencyHigh([]string{"low"}) {
		t.Error("expected low not flagged")
	}
	if !IsSaliencyKnown([]string{"known_pattern"}) {
		t.Error("expected known_pattern flagged")
	}
	if IsSaliencyKnown(nil) {
		t.Error("expected empty saliency not known")
	}
}
