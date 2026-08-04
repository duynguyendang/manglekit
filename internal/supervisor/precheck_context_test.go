package supervisor

import (
	"context"
	"strings"
	"testing"

	"github.com/duynguyendang/manglekit/core"
)

// recordingEvaluator embeds passThroughEvaluator and captures the
// envelope AssessPlan receives, optionally halting on a reason.
type recordingEvaluator struct {
	passThroughEvaluator
	captured *core.Envelope
	halt     bool
}

func (r *recordingEvaluator) AssessPlan(ctx context.Context, input core.Envelope) (core.Decision, error) {
	cp := input
	r.captured = &cp
	if r.halt {
		return core.Decision{Outcome: core.DecisionHalt, Reasons: []string{"test halt"}}, nil
	}
	return core.Decision{Outcome: core.DecisionProceed}, nil
}

// TestSupervisedV2_PreCheckSeesRequestContext verifies that the
// supervised pre-flight check receives the caller's action name,
// metadata, security labels, and explicit facts — not just the
// payload-derived quads. This is the contract policies gated on
// action_operation/2, meta/2, or label/1 depend on.
func TestSupervisedV2_PreCheckSeesRequestContext(t *testing.T) {
	eval := &recordingEvaluator{}
	inner := &mockSDKAction{}
	sv := NewSupervisedActionFromSDK(inner, eval, core.NopLogger{})

	env := core.NewEnvelope("payload text")
	env.Metadata["user"] = "alice"
	env.Metadata["memory_hit_0"] = "doc_project_x"
	env.SecurityLabels = []string{"tainted"}
	env.Facts = append(env.Facts, `type("Req", "query").`)

	if _, err := sv.Execute(context.Background(), env); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if eval.captured == nil {
		t.Fatal("pre-check AssessPlan was never invoked")
	}
	got := eval.captured

	if got.Metadata["user"] != "alice" || got.Metadata["memory_hit_0"] != "doc_project_x" {
		t.Errorf("pre-check envelope missing caller metadata, got: %v", got.Metadata)
	}
	if len(got.SecurityLabels) != 1 || got.SecurityLabels[0] != "tainted" {
		t.Errorf("pre-check envelope missing security labels, got: %v", got.SecurityLabels)
	}
	joined := strings.Join(got.Facts, "\n")
	if !strings.Contains(joined, `type("Req", "query").`) {
		t.Errorf("pre-check envelope missing caller facts, got: %v", got.Facts)
	}
	if !strings.Contains(joined, `action_operation("Req", "mock_sdk_action").`) {
		t.Errorf("pre-check envelope missing action_operation fact, got: %v", got.Facts)
	}
}

// TestSupervisedV2_PreCheckHaltBlocksExecution verifies that a HALT
// from the policy pre-check blocks the inner action: the supervised
// path enforces, not just observes.
func TestSupervisedV2_PreCheckHaltBlocksExecution(t *testing.T) {
	eval := &recordingEvaluator{halt: true}
	inner := &mockSDKAction{}
	sv := NewSupervisedActionFromSDK(inner, eval, core.NopLogger{})

	env := core.NewEnvelope("payload text")
	env.Metadata["user"] = "mallory"

	_, err := sv.Execute(context.Background(), env)
	if err == nil {
		t.Fatal("expected pre-check halt to surface as an error")
	}
	if inner.capturedPayload != nil {
		t.Fatal("inner action executed despite policy HALT — supervision did not block")
	}
}

// TestSupervisedV2_NoContextStillPasses pins the legacy behavior: with
// no metadata/labels/facts and a payload that yields no quads, the
// pre-check still runs (action_operation is always present) and a
// permissive policy proceeds.
func TestSupervisedV2_NoContextStillPasses(t *testing.T) {
	eval := &recordingEvaluator{}
	inner := &mockSDKAction{}
	sv := NewSupervisedActionFromSDK(inner, eval, core.NopLogger{})

	if _, err := sv.Execute(context.Background(), core.NewEnvelope("plain text")); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if inner.capturedPayload == nil {
		t.Fatal("inner action did not execute")
	}
}
