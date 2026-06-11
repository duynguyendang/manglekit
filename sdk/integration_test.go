package sdk

import (
	"context"
	"testing"

	"github.com/duynguyendang/manglekit/core"
	function "github.com/duynguyendang/manglekit/adapters/func"
	"github.com/duynguyendang/manglekit/internal/engine"
	"github.com/duynguyendang/manglekit/internal/supervisor"
	"github.com/stretchr/testify/require"
)

// TestPreCheckHaltOnActionOperation asserts that a policy gating on
// action_operation/2 blocks the matching action.
func TestPreCheckHaltOnActionOperation(t *testing.T) {
	eval, err := engine.New()
	require.NoError(t, err)

	policy := `halt("Req", "blocked_action") :- action_operation("Req", "blocked_action").`
	require.NoError(t, eval.LoadPolicy(context.Background(), policy))

	inner := function.New("blocked_action", func(_ context.Context, _ string) (string, error) {
		t.Fatal("inner action should not execute")
		return "", nil
	})

	sv := supervisor.NewSupervisedActionFromSDK(inner, eval, "closed", core.NopLogger{})

	_, err = sv.Execute(context.Background(), core.NewEnvelope("payload"))
	require.Error(t, err, "policy should halt on action_operation match")
	require.True(t, core.IsPolicyViolationError(err), "error should be a policy violation")
	require.Contains(t, err.Error(), "blocked_action")
}

// TestPreCheckProceedOnActionOperation asserts that a policy gating on
// a specific action name does NOT block a different action.
func TestPreCheckProceedOnActionOperation(t *testing.T) {
	eval, err := engine.New()
	require.NoError(t, err)

	policy := `halt("Req", "blocked_action") :- action_operation("Req", "blocked_action").`
	require.NoError(t, eval.LoadPolicy(context.Background(), policy))

	executed := false
	inner := function.New("safe_action", func(_ context.Context, _ string) (string, error) {
		executed = true
		return "ok", nil
	})

	sv := supervisor.NewSupervisedActionFromSDK(inner, eval, "closed", core.NopLogger{})

	_, err = sv.Execute(context.Background(), core.NewEnvelope("payload"))
	require.NoError(t, err, "policy should allow non-matching action")
	require.True(t, executed, "inner action should have executed")
}

// TestPreCheckHaltOnMeta asserts that a policy gating on meta/2 blocks
// when the metadata matches.
func TestPreCheckHaltOnMeta(t *testing.T) {
	eval, err := engine.New()
	require.NoError(t, err)

	policy := `halt("Req", "banned_role") :- meta("role", "banned").`
	require.NoError(t, eval.LoadPolicy(context.Background(), policy))

	inner := function.New("test_action", func(_ context.Context, _ string) (string, error) {
		t.Fatal("inner action should not execute")
		return "", nil
	})

	sv := supervisor.NewSupervisedActionFromSDK(inner, eval, "closed", core.NopLogger{})

	env := core.NewEnvelope("payload")
	env.Metadata["role"] = "banned"

	_, err = sv.Execute(context.Background(), env)
	require.Error(t, err, "policy should halt on meta match")
	require.True(t, core.IsPolicyViolationError(err))
}

// TestPreCheckHaltOnLabel asserts that a policy gating on label/1 blocks
// when a security label matches.
func TestPreCheckHaltOnLabel(t *testing.T) {
	eval, err := engine.New()
	require.NoError(t, err)

	policy := `halt("Req", "classified_label") :- label("classified").`
	require.NoError(t, eval.LoadPolicy(context.Background(), policy))

	inner := function.New("test_action", func(_ context.Context, _ string) (string, error) {
		t.Fatal("inner action should not execute")
		return "", nil
	})

	sv := supervisor.NewSupervisedActionFromSDK(inner, eval, "closed", core.NopLogger{})

	env := core.NewEnvelope("payload")
	env.SecurityLabels = []string{"classified"}

	_, err = sv.Execute(context.Background(), env)
	require.Error(t, err, "policy should halt on label match")
	require.True(t, core.IsPolicyViolationError(err))
}

// TestPreCheckHaltOnPayloadFact asserts that a policy gating on
// mangle-tagged payload fields blocks when the field matches.
// The mangle tag generates facts via flattenToQuads which uses the
// predicate from the tag. Since meta/2 is a stdlib predicate, we use
// it to simulate payload reflection via metadata.
func TestPreCheckHaltOnPayloadFact(t *testing.T) {
	eval, err := engine.New()
	require.NoError(t, err)

	// Use meta/2 to test payload-derived facts — this is what the
	// supervisor actually injects when CustomHybridMemory provides
	// metadata from mangle-tagged fields.
	policy := `halt("Req", "dangerous_input") :- meta("input_type", "dangerous").`
	require.NoError(t, eval.LoadPolicy(context.Background(), policy))

	inner := function.New("test_action", func(_ context.Context, _ string) (string, error) {
		t.Fatal("inner action should not execute")
		return "", nil
	})

	sv := supervisor.NewSupervisedActionFromSDK(inner, eval, "closed", core.NopLogger{})

	env := core.NewEnvelope("payload")
	env.Metadata["input_type"] = "dangerous"

	_, err = sv.Execute(context.Background(), env)
	require.Error(t, err, "policy should halt on payload-derived fact match")
	require.True(t, core.IsPolicyViolationError(err))
}

// TestPreCheckHaltOnExplicitFact asserts that a policy gating on
// caller-supplied facts blocks when the fact matches.
// Since Datalog requires Decl for all predicates, the fact uses
// a predicate that the policy itself declares.
func TestPreCheckHaltOnExplicitFact(t *testing.T) {
	eval, err := engine.New()
	require.NoError(t, err)

	// The policy must declare source/2 so the analyzer accepts it.
	policy := `
Decl source(E, V).
halt("Req", "untrusted_source") :- source("Req", "untrusted").
`
	require.NoError(t, eval.LoadPolicy(context.Background(), policy))

	inner := function.New("test_action", func(_ context.Context, _ string) (string, error) {
		t.Fatal("inner action should not execute")
		return "", nil
	})

	sv := supervisor.NewSupervisedActionFromSDK(inner, eval, "closed", core.NopLogger{})

	env := core.NewEnvelope("payload")
	env.Facts = append(env.Facts, `source("Req", "untrusted").`)

	_, err = sv.Execute(context.Background(), env)
	require.Error(t, err, "policy should halt on explicit fact match")
	require.True(t, core.IsPolicyViolationError(err))
}

// TestPreCheckBlocksInnerAction verifies that a HALT from the pre-check
// prevents the inner action from executing at all.
func TestPreCheckBlocksInnerAction(t *testing.T) {
	eval, err := engine.New()
	require.NoError(t, err)

	policy := `halt("Req", "always_block").`
	require.NoError(t, eval.LoadPolicy(context.Background(), policy))

	innerExecuted := false
	inner := function.New("test_action", func(_ context.Context, _ string) (string, error) {
		innerExecuted = true
		return "ok", nil
	})

	sv := supervisor.NewSupervisedActionFromSDK(inner, eval, "closed", core.NopLogger{})

	_, err = sv.Execute(context.Background(), core.NewEnvelope("payload"))
	require.Error(t, err)
	require.False(t, innerExecuted, "inner action must not execute when pre-check halts")
}

// TestPreCheckAuditTrailPopulated verifies that when the pre-check halts,
// the audit trail is populated with the matched rules.
func TestPreCheckAuditTrailPopulated(t *testing.T) {
	eval, err := engine.New()
	require.NoError(t, err)

	policy := `halt("Req", "audit_test") :- meta("trigger", "yes").`
	require.NoError(t, eval.LoadPolicy(context.Background(), policy))

	inner := function.New("test_action", func(_ context.Context, _ string) (string, error) {
		return "ok", nil
	})

	sv := supervisor.NewSupervisedActionFromSDK(inner, eval, "closed", core.NopLogger{})

	env := core.NewEnvelope("payload")
	env.Metadata["trigger"] = "yes"

	_, err = sv.Execute(context.Background(), env)
	require.Error(t, err)
	require.True(t, core.IsPolicyViolationError(err))
	// The error string should contain the conflict path from the policy
	require.Contains(t, err.Error(), "audit_test")
}

// TestPostCheckHaltOnOutput verifies that the Reflect (post-check) path
// halts when the output violates the policy.
func TestPostCheckHaltOnOutput(t *testing.T) {
	eval, err := engine.New()
	require.NoError(t, err)

	// Policy that blocks any output containing a "blocked" field.
	// Declare content/2 so the analyzer accepts it as an EDB predicate.
	policy := `
Decl content(E, V).
halt("Output", "blocked_output") :- content("Output", "blocked").
`
	require.NoError(t, eval.LoadPolicy(context.Background(), policy))

	type output struct {
		Content string `mangle:"content"`
	}

	inner := function.New("test_action", func(_ context.Context, _ string) (output, error) {
		return output{Content: "blocked"}, nil
	})

	sv := supervisor.NewSupervisedActionFromSDK(inner, eval, "closed", core.NopLogger{})

	_, err = sv.Execute(context.Background(), core.NewEnvelope("payload"))
	require.Error(t, err, "post-check should halt on output policy violation")
	require.True(t, core.IsPolicyViolationError(err))
	require.Contains(t, err.Error(), "blocked")
}

// TestPostCheckProceedOnCleanOutput verifies that the Reflect (post-check)
// passes when the output does not violate the policy.
func TestPostCheckProceedOnCleanOutput(t *testing.T) {
	eval, err := engine.New()
	require.NoError(t, err)

	// Policy that only blocks output containing "blocked"
	policy := `
Decl content(E, V).
halt("Output", "blocked_output") :- content("Output", "blocked").
`
	require.NoError(t, eval.LoadPolicy(context.Background(), policy))

	executed := false
	type output struct {
		Content string `mangle:"content"`
	}

	inner := function.New("test_action", func(_ context.Context, _ string) (output, error) {
		executed = true
		return output{Content: "safe"}, nil
	})

	sv := supervisor.NewSupervisedActionFromSDK(inner, eval, "closed", core.NopLogger{})

	_, err = sv.Execute(context.Background(), core.NewEnvelope("payload"))
	require.NoError(t, err, "post-check should pass on clean output")
	require.True(t, executed)
}

// TestPreCheckProceedWithNoMetadata asserts that a supervised action
// executes normally when no metadata/labels/facts are provided and the
// policy is permissive.
func TestPreCheckProceedWithNoMetadata(t *testing.T) {
	eval, err := engine.New()
	require.NoError(t, err)

	// Permissive policy: halt only on meta("role", "admin")
	policy := `halt("Req", "admin_blocked") :- meta("role", "admin").`
	require.NoError(t, eval.LoadPolicy(context.Background(), policy))

	executed := false
	inner := function.New("test_action", func(_ context.Context, _ string) (string, error) {
		executed = true
		return "ok", nil
	})

	sv := supervisor.NewSupervisedActionFromSDK(inner, eval, "closed", core.NopLogger{})

	_, err = sv.Execute(context.Background(), core.NewEnvelope("payload"))
	require.NoError(t, err)
	require.True(t, executed)
}
