package supervisor

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/core/domain"
)

// fakeEvaluator implements core.Evaluator for adapter-level tests.
type fakeEvaluator struct {
	assessPlan func(ctx context.Context, input core.Envelope) (core.Decision, error)
	reflect    func(ctx context.Context, meta core.ActionMetadata, out core.Envelope) (core.Envelope, error)
}

func (f *fakeEvaluator) AssessPlan(ctx context.Context, input core.Envelope) (core.Decision, error) {
	if f.assessPlan != nil {
		return f.assessPlan(ctx, input)
	}
	return core.Decision{Outcome: core.DecisionProceed}, nil
}

func (f *fakeEvaluator) Assess(ctx context.Context, meta core.ActionMetadata, input core.Envelope) error {
	return nil
}

func (f *fakeEvaluator) Reflect(ctx context.Context, meta core.ActionMetadata, out core.Envelope) (core.Envelope, error) {
	if f.reflect != nil {
		return f.reflect(ctx, meta, out)
	}
	return out, nil
}

func (f *fakeEvaluator) EvaluateSteering(ctx context.Context, input core.Envelope) (string, map[string]string, error) {
	return "", nil, nil
}

func (f *fakeEvaluator) GetActionConfig(ctx context.Context, input core.Envelope) (map[string]string, error) {
	return nil, nil
}

func (f *fakeEvaluator) CheckRequirement(ctx context.Context, input core.Envelope, reqName string) (bool, error) {
	return false, nil
}

func (f *fakeEvaluator) RegisterExternalPredicate(name string, fn func(ctx context.Context, inputs []any) ([][]any, error)) error {
	return nil
}

func (f *fakeEvaluator) LoadPolicy(ctx context.Context, source string) error { return nil }

func (f *fakeEvaluator) LoadFromSource(ctx context.Context, source string) error { return nil }

func (f *fakeEvaluator) LoadFacts(ctx context.Context, facts []string) error { return nil }

func (f *fakeEvaluator) RegisterAction(meta core.ActionMetadata) error { return nil }

func (f *fakeEvaluator) Query(ctx context.Context, facts []string, queryStr string) ([]map[string]string, error) {
	return nil, nil
}

func (f *fakeEvaluator) Logger() core.Logger { return core.NopLogger{} }

func haltTrail(tiers ...core.Tier) *core.AuditTrail {
	trail := core.NewAuditTrail("fake", "halt-query")
	for _, t := range tiers {
		trail.AddRule("halt", fmt.Sprintf(`halt("Req", "r", "%s")`, t), "policy.dl", "halt", t, nil)
	}
	return trail
}

func reqCtx(entity string) context.Context {
	return withPreCheckContext(context.Background(), &preCheckContext{
		actionName: "test-action",
		entityID:   entity,
	})
}

var probeAtom = []domain.Atom{{Subject: "Req", Predicate: "query", Object: "hello"}}

// P0.7: the SDK adapter must carry the engine decision's REAL tier, so the
// gate can apply tier semantics instead of blocking everything as Tier1.
func TestVerifyAtoms_HaltCarriesRealTier(t *testing.T) {
	cases := []struct {
		name   string
		trail  *core.AuditTrail
		reason string
		want   domain.TrustTier
	}{
		{"T0 axiom blocks", haltTrail(core.TierT0_Axiom), "kernel law", domain.Tier0Kernel},
		{"T1 governance blocks", haltTrail(core.TierT1_Governance), "admin policy", domain.Tier1Admin},
		{"T2 playbook is soft", haltTrail(core.TierT2_Playbook), "advisory", domain.Tier2AI},
		{"T3 user is soft", haltTrail(core.TierT3_User), "user hint", domain.Tier3User},
		{"mixed takes most severe", haltTrail(core.TierT3_User, core.TierT1_Governance), "mixed", domain.Tier1Admin},
		{"unknown tier stays hard", haltTrail(core.TierUnknown), "legacy", domain.Tier1Admin},
		{"no trail stays hard", nil, "legacy deny", domain.Tier1Admin},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := &sdkEvaluatorAdapter{inner: &fakeEvaluator{
				assessPlan: func(context.Context, core.Envelope) (core.Decision, error) {
					return core.Decision{
						Outcome:    core.DecisionHalt,
						Reasons:    []string{tc.reason},
						AuditTrail: tc.trail,
					}, nil
				},
			}}
			res, err := a.VerifyAtoms(reqCtx(core.EntityInput), probeAtom, nil)
			if err != nil {
				t.Fatalf("policy deny must not surface as Go error, got %v", err)
			}
			if res.Pass {
				t.Fatal("expected Pass=false on halt")
			}
			if res.ViolationTier != tc.want {
				t.Errorf("ViolationTier = %v, want %v", res.ViolationTier, tc.want)
			}
			if res.ConflictPath != tc.reason {
				t.Errorf("ConflictPath = %q, want %q", res.ConflictPath, tc.reason)
			}
		})
	}
}

// Engine failures (not policy denies) must surface as Go errors so the gate
// wraps them in core.SupervisorError — fail-closed but distinguishable.
func TestVerifyAtoms_EngineErrorSurfacesAsError(t *testing.T) {
	engineErr := errors.New("halt query error: boom")
	a := &sdkEvaluatorAdapter{inner: &fakeEvaluator{
		assessPlan: func(context.Context, core.Envelope) (core.Decision, error) {
			return core.Decision{Outcome: core.DecisionHalt, Reasons: []string{"boom"}}, engineErr
		},
	}}
	_, err := a.VerifyAtoms(reqCtx(core.EntityInput), probeAtom, nil)
	if err == nil {
		t.Fatal("expected Go error for engine failure")
	}
	if !errors.Is(err, engineErr) {
		t.Errorf("error must wrap the engine error, got %v", err)
	}

	// And the gate turns it into a SupervisorError (blocks).
	inner := &mockAction{executeResult: domain.Envelope{Payload: "x"}}
	sup := New(inner, a, &mockGenePoolPort{})
	_, err = sup.ExecuteInternal(reqCtx(core.EntityInput), "test-intent", core.Envelope{Payload: "p"})
	if err == nil || !errors.Is(err, core.ErrSupervisorFailure) {
		t.Fatalf("expected ErrSupervisorFailure from gate, got %v", err)
	}
}

func TestVerifyAtoms_PostCheckAlignmentCarriesTier(t *testing.T) {
	a := &sdkEvaluatorAdapter{inner: &fakeEvaluator{
		reflect: func(_ context.Context, _ core.ActionMetadata, _ core.Envelope) (core.Envelope, error) {
			return core.Envelope{}, &core.AlignmentError{
				Message: "output flagged",
				Tier:    core.TierT2_Playbook,
			}
		},
	}}
	res, err := a.VerifyAtoms(reqCtx(core.EntityOutput), probeAtom, nil)
	if err != nil {
		t.Fatalf("alignment deny must not surface as Go error, got %v", err)
	}
	if res.Pass || res.ViolationTier != domain.Tier2AI {
		t.Errorf("expected soft T2 deny, got pass=%v tier=%v", res.Pass, res.ViolationTier)
	}

	// Non-alignment failures return an error (gate → SupervisorError).
	b := &sdkEvaluatorAdapter{inner: &fakeEvaluator{
		reflect: func(context.Context, core.ActionMetadata, core.Envelope) (core.Envelope, error) {
			return core.Envelope{}, errors.New("policy evaluation error")
		},
	}}
	if _, err := b.VerifyAtoms(reqCtx(core.EntityOutput), probeAtom, nil); err == nil {
		t.Fatal("expected Go error for post-check engine failure")
	}
}

// P0.5 residual: a payload producing an unbounded number of facts must be
// refused fail-closed instead of flooding the gate evaluation.
func TestVerifyAtoms_FactCapBlocks(t *testing.T) {
	atoms := make([]domain.Atom, maxFactsPerVerify+1)
	for i := range atoms {
		atoms[i] = domain.Atom{Subject: "Req", Predicate: fmt.Sprintf("p%d", i), Object: "v"}
	}
	a := &sdkEvaluatorAdapter{inner: &fakeEvaluator{}}
	res, err := a.VerifyAtoms(reqCtx(core.EntityInput), atoms, nil)
	if err != nil {
		t.Fatalf("expected AuditResult deny, not Go error: %v", err)
	}
	if res.Pass || res.ViolationTier != domain.Tier0Kernel {
		t.Errorf("expected Tier0 block, got pass=%v tier=%v", res.Pass, res.ViolationTier)
	}
	if !strings.Contains(res.ConflictPath, "fact_limit_exceeded") {
		t.Errorf("ConflictPath = %q, want fact_limit_exceeded", res.ConflictPath)
	}

	// Just under the cap: no block from the cap itself.
	res2, err := a.VerifyAtoms(reqCtx(core.EntityInput), atoms[:maxFactsPerVerify], nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res2.Pass {
		t.Error("facts at the cap limit must not trip the cap")
	}
}

// Soft-tier semantics must be symmetric: a post-check deny at T2/T3 does
// not block the assembled output; verifier errors still do (ADR-001).
func TestSupervisedAction_PostCheck_SoftTierDoesNotBlock(t *testing.T) {
	inner := &mockAction{executeResult: domain.Envelope{Payload: "result"}}
	verifier := &multiCallReasoningPort{
		results: []*domain.AuditResult{
			{Pass: true},                                                        // pre-check
			{Pass: false, ViolationTier: domain.Tier2AI, ConflictPath: "soft"}, // post-check soft
		},
		errs: []error{nil, nil},
	}
	sup := New(inner, verifier, &mockGenePoolPort{})
	result, err := sup.ExecuteInternal(postCheckTestEnv(context.Background()), "intent", core.Envelope{Payload: "in"})
	if err != nil {
		t.Fatalf("soft post-check tier must not block, got %v", err)
	}
	if result.Payload != "result" {
		t.Errorf("unexpected payload %v", result.Payload)
	}
}
