package engine

import (
	"context"
	"testing"

	"github.com/duynguyendang/manglekit/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAssessDenyErrorCarriesStructuredFields asserts that a policy-deny
// AlignmentError carries the action name, matched rule text, and tier from
// the gate evaluation (T-007).
func TestAssessDenyErrorCarriesStructuredFields(t *testing.T) {
	pe, err := New()
	require.NoError(t, err)

	policy := `halt("Req", "n too big", "T1") :- meta("n", "20").`
	require.NoError(t, pe.LoadPolicy(context.Background(), policy))

	env := core.NewEnvelope("payload")
	env.Metadata["n"] = "20"

	err = pe.Assess(context.Background(), core.ActionMetadata{Name: "my_action"}, env)
	require.Error(t, err)

	var alignErr *core.AlignmentError
	require.ErrorAs(t, err, &alignErr)
	assert.Equal(t, "my_action", alignErr.ActionName)
	assert.Equal(t, core.TierT1_Governance, alignErr.Tier)
	assert.Contains(t, alignErr.MatchedRule, `halt("Req", "n too big", "T1")`)
	assert.Equal(t, "n too big", alignErr.Message)
	// The block-detection idiom works on the Assess path too.
	assert.True(t, core.IsPolicyViolationError(err))
}

// TestAddPolicyAutoDeclaresExternalPredicates asserts that AddPolicy (the
// LoadPolicy route) auto-emits `Decl ... external()` for registered external
// predicates, like LoadFromSource (T-008).
func TestAddPolicyAutoDeclaresExternalPredicates(t *testing.T) {
	runtime := NewMangleRuntime()

	err := runtime.RegisterExternalPredicate("double", func(_ context.Context, inputs []any) ([][]any, error) {
		n := inputs[0].(int64)
		return [][]any{{n * 2}}, nil
	})
	require.NoError(t, err)

	// AddPolicy (not LoadFromSource) — must not fail with
	// "ext callback for predicate ... not marked as external()".
	policy := `result(X) :- double(5, X).`
	err = runtime.AddPolicy(context.Background(), policy)
	require.NoError(t, err)

	ok, err := runtime.ExecuteQuery(context.Background(), nil, "result(10)")
	require.NoError(t, err)
	assert.True(t, ok, "result(10) should be derivable via the external predicate")
}

// TestAddPolicySkipsUnreferencedExternalPredicates asserts that external
// predicates registered but not referenced by the policy do not produce
// spurious declarations.
func TestAddPolicySkipsUnreferencedExternalPredicates(t *testing.T) {
	runtime := NewMangleRuntime()

	err := runtime.RegisterExternalPredicate("unused_ext", func(_ context.Context, inputs []any) ([][]any, error) {
		return nil, nil
	})
	require.NoError(t, err)

	err = runtime.AddPolicy(context.Background(), `base("a"). ok_rule(X) :- base(X).`)
	require.NoError(t, err)
}
