package oodaflow

import (
	"context"
	"strings"
	"testing"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/sdk"
	"github.com/duynguyendang/manglekit/x/ooda"
)

// TestDefaultsAreSingleSourced pins the 0.8 parity contract (ADR-004 T-1805):
// the sdk config-side default (unexported const behind Client.ParadoxThreshold)
// and the x/ooda frame-level default must agree, and x/oodaflow must take its
// default from x/ooda rather than a local literal. This test is the only place
// allowed to see both sides of the mirror.
func TestDefaultsAreSingleSourced(t *testing.T) {
	ctx := context.Background()
	client, err := sdk.NewClient(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = client.Shutdown(ctx) }()

	if got := client.ParadoxThreshold(); got != ooda.DefaultParadoxThreshold {
		t.Errorf("sdk client default = %v, want parity with x/ooda %v", got, ooda.DefaultParadoxThreshold)
	}
	flow := NewOODAFlow(&OODAFlowConfig{})
	if flow.config.ParadoxThreshold != ooda.DefaultParadoxThreshold {
		t.Errorf("oodaflow default = %v, want %v (sourced from x/ooda)",
			flow.config.ParadoxThreshold, ooda.DefaultParadoxThreshold)
	}
}

// echoBrain is a minimal mock: always proceed with the echo tool action.
type echoBrain struct {
	decision *core.Decision
}

func (b echoBrain) Evaluate(context.Context, *ooda.CognitiveFrame) (*core.Decision, error) {
	return b.decision, nil
}

func (b echoBrain) Verify(context.Context, *ooda.CognitiveFrame) (*core.AuditTrail, error) {
	return &core.AuditTrail{}, nil
}

func (b echoBrain) LoadPolicy(context.Context, string) error { return nil }

func TestOODAFlowRunSmoke(t *testing.T) {
	registry := ooda.NewRegistry()
	registry.MustRegister("echo", func(_ context.Context, args map[string]any) (string, error) {
		in, _ := args["input"].(string)
		return "echo:" + in, nil
	})

	flow := NewOODAFlow(&OODAFlowConfig{MaxRetries: 2, Timeout: 0})
	// The flow builds its own frame per Run; wire components through the
	// config-compatible fields (Brain/Dispatcher) to keep the mock offline.
	flow.config.Brain = echoBrain{decision: &core.Decision{
		Outcome: core.DecisionProceed,
		Action:  core.NewActionEnvelope("echo", nil),
	}}
	flow.config.Dispatcher = ooda.NewDispatcher(registry)

	out, err := flow.Run(context.Background(), &OODAFlowInput{Input: "hello"})
	if err != nil {
		t.Fatalf("flow.Run: %v", err)
	}
	if out.Error != "" {
		t.Fatalf("flow reported error: %s", out.Error)
	}
	if !strings.HasPrefix(out.Output, "echo:") {
		t.Errorf("Output = %q, want echo: prefix", out.Output)
	}
	if out.Status != ooda.VerifyStatusPassed {
		t.Errorf("Status = %s, want passed", out.Status)
	}
}

func TestFlowRegistryRoundTrip(t *testing.T) {
	reg := NewFlowRegistry(nil)
	flow := NewOODAFlow(&OODAFlowConfig{})
	reg.Register("basic", flow)
	if got, ok := reg.Get("basic"); !ok || got != flow {
		t.Fatal("registry lookup failed")
	}
	if _, err := reg.Run(context.Background(), "missing", &OODAFlowInput{}); err == nil {
		t.Fatal("expected error for unknown flow")
	}
}
