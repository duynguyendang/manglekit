package engine

import (
	"context"
	"strings"
	"testing"

	"github.com/duynguyendang/manglekit/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const explainPolicy = `
suspicious("Req").
risky(Req) :- suspicious(Req).
violation_msg("prompt injection detected") :- risky(Req).
halt(Req, M, "T1") :- risky(Req), violation_msg(M).
`

func TestExplain_DerivationTree(t *testing.T) {
	e, err := New()
	require.NoError(t, err)
	require.NoError(t, e.LoadPolicy(context.Background(), explainPolicy))

	expl, err := e.Explain(context.Background(), nil, `halt("Req", Reason, Tier)`)
	require.NoError(t, err)
	require.True(t, expl.Outcome)
	require.Len(t, expl.Derivations, 1)

	root := expl.Derivations[0]
	assert.Equal(t, core.ViaRule, root.Via)
	assert.Equal(t, core.TierT1_Governance, root.Tier, "tier must come from the actual rule instantiation, not heuristics")
	assert.Equal(t, "halt", root.Predicate)
	assert.Contains(t, root.Fact, `"prompt injection detected"`)
	assert.Contains(t, root.Rule, ":- risky(Req), violation_msg(M)")
	assert.NotEmpty(t, root.Bindings["M"], "variable bindings should be grounded")

	// Two premise derivations: risky("Req") (rule) and violation_msg(...) (rule).
	require.Len(t, root.Children, 2)
	assert.Equal(t, "risky", root.Children[0].Predicate)
	assert.Equal(t, core.ViaRule, root.Children[0].Via)
	assert.Equal(t, "violation_msg", root.Children[1].Predicate)

	// The risky("Req") hop itself traces down to the base fact suspicious("Req").
	require.Len(t, root.Children[0].Children, 1)
	assert.Equal(t, "suspicious", root.Children[0].Children[0].Predicate)
	assert.Equal(t, core.ViaFact, root.Children[0].Children[0].Via)
}

func TestExplain_NoMatch(t *testing.T) {
	e, err := New()
	require.NoError(t, err)
	require.NoError(t, e.LoadPolicy(context.Background(), explainPolicy))

	expl, err := e.Explain(context.Background(), nil, `halt("Other", Reason, Tier)`)
	require.NoError(t, err)
	assert.False(t, expl.Outcome)
	assert.Empty(t, expl.Derivations)
}

func TestExplain_BaseFactDirect(t *testing.T) {
	e, err := New()
	require.NoError(t, err)
	require.NoError(t, e.LoadFacts(context.Background(), []string{`flag("on").`}))

	expl, err := e.Explain(context.Background(), nil, `flag(X)`)
	require.NoError(t, err)
	require.True(t, expl.Outcome)
	require.Len(t, expl.Derivations, 1)
	assert.Equal(t, core.ViaFact, expl.Derivations[0].Via)
	assert.Equal(t, `flag("on")`, expl.Derivations[0].Fact)
}

func TestExplain_TemporaryFacts(t *testing.T) {
	e, err := New()
	require.NoError(t, err)
	require.NoError(t, e.LoadPolicy(context.Background(), explainPolicy))

	// Extra request-scoped fact participates in the derivation.
	expl, err := e.Explain(context.Background(), []string{`violation_msg("extra reason").`}, `halt("Req", Reason, Tier)`)
	require.NoError(t, err)
	assert.True(t, expl.Outcome)
	found := false
	for _, d := range expl.Derivations {
		if strings.Contains(d.Fact, "extra reason") {
			found = true
		}
	}
	assert.True(t, found, "derivation over temporary facts should be explained")
}

func TestExplain_AuditTrailCompatibility(t *testing.T) {
	e, err := New()
	require.NoError(t, err)
	require.NoError(t, e.LoadPolicy(context.Background(), explainPolicy))

	expl, err := e.Explain(context.Background(), nil, `halt("Req", Reason, Tier)`)
	require.NoError(t, err)

	trail := expl.AuditTrail("manglekit-engine")
	require.NotNil(t, trail)
	assert.Equal(t, 1, trail.MatchedCount)
	require.NotEmpty(t, trail.MatchedRules)
	assert.Equal(t, "halt", trail.MatchedRules[0].RuleName)
	assert.Equal(t, core.TierT1_Governance, trail.MatchedRules[0].Tier)
	assert.Contains(t, trail.MatchedRules[0].Definition, ":- risky(Req), violation_msg(M)")
	assert.NotEmpty(t, trail.MatchedRules[0].Bindings)
}

func TestQueryWithAudit_RealProvenance(t *testing.T) {
	e, err := New()
	require.NoError(t, err)
	require.NoError(t, e.LoadPolicy(context.Background(), explainPolicy))

	res, err := e.QueryWithAudit(context.Background(), nil, `halt("Req", Reason, Tier)`)
	require.NoError(t, err)
	require.Len(t, res.Results, 1)
	require.NotEmpty(t, res.AuditTrail.MatchedRules)

	rule := res.AuditTrail.MatchedRules[0]
	assert.Equal(t, core.TierT1_Governance, rule.Tier, "tier provenance from the real rule, not TierMapping")
	assert.Contains(t, rule.Definition, ":- risky(Req), violation_msg(M)", "definition is the real rule text")
	assert.NotEqual(t, "governance.dl", rule.SourceFile, "no filename heuristic")
}

// comparisonPolicy uses an integer comparison builtin in the rule body — the
// shape a numeric policy gate needs (observed value vs a policy-owned bar).
const comparisonPolicy = `
Decl policy_anchor(Marker).
policy_anchor("ok").

Decl class_bar(Class, Bar).
class_bar("account_access", 950) :- policy_anchor("ok").

Decl choice(Option).
Decl score_pm(Option, Score).

halt("Req", "below_bar", "T1") :-
    choice(C), score_pm(C, P), class_bar(C, T), :lt(P, T).
`

// TestExplain_ComparisonBuiltinIsEvaluatedNotMislabelled is a regression guard
// for the audit trail: :lt/:le/:gt/:ge parse as atoms with a ":"-prefixed
// symbol, but no fact of that shape ever exists in the store. Before the fix
// the prover treated them as ordinary predicates, the whole rule failed to
// reconstruct, and the derived halt was reported as "[base fact]" — hiding the
// arithmetic that actually fired.
func TestExplain_ComparisonBuiltinIsEvaluatedNotMislabelled(t *testing.T) {
	e, err := New()
	require.NoError(t, err)
	require.NoError(t, e.LoadPolicy(context.Background(), comparisonPolicy))

	facts := []string{
		`choice("account_access").`,
		`score_pm("account_access", 900).`,
	}

	expl, err := e.Explain(context.Background(), facts, `halt("Req", Reason, Tier)`)
	require.NoError(t, err)
	require.True(t, expl.Outcome, "900 < 950 must derive the halt")
	require.Len(t, expl.Derivations, 1)

	root := expl.Derivations[0]
	assert.Equal(t, core.ViaRule, root.Via, "a rule-derived fact must not be reported as a base fact")
	assert.Contains(t, root.Rule, ":lt(P,T)")

	// The comparison itself must appear as an evaluated builtin step, with the
	// grounded operands — that is the audit value.
	var builtin *core.DerivationStep
	for i := range root.Children {
		if root.Children[i].Via == core.ViaBuiltin {
			builtin = &root.Children[i]
		}
	}
	require.NotNil(t, builtin, "proof tree must contain the evaluated comparison; got:\n%s", expl.String())
	assert.Equal(t, "900 < 950", builtin.Fact)

	// The operands must also be grounded as their own hops.
	joined := expl.String()
	assert.Contains(t, joined, `score_pm("account_access",900)`)
	assert.Contains(t, joined, `class_bar("account_access",950)`)
}

// TestExplain_ComparisonBuiltinFalseBranchDoesNotProve: when the comparison
// fails, the rule must NOT be reported as firing.
func TestExplain_ComparisonBuiltinFalseBranchDoesNotProve(t *testing.T) {
	e, err := New()
	require.NoError(t, err)
	require.NoError(t, e.LoadPolicy(context.Background(), comparisonPolicy))

	expl, err := e.Explain(context.Background(), []string{
		`choice("account_access").`,
		`score_pm("account_access", 970).`, // clears the 950 bar
	}, `halt("Req", Reason, Tier)`)
	require.NoError(t, err)
	assert.False(t, expl.Outcome, "970 >= 950 must not derive a halt; tree:\n%s", expl.String())
}

// TestExplain_ComparisonBuiltinRejectsNonInteger pins WHY numeric policy gates
// must scale to integers: Mangle's :lt/:le/:gt/:ge are integer-only
// (builtin.getNumberValue requires ast.NumberType), and a float operand fails
// the evaluation loudly — "value 0.9 (4) is not a number" — instead of being
// silently coerced. Callers scale probabilities to per-mille/basis points.
func TestExplain_ComparisonBuiltinRejectsNonInteger(t *testing.T) {
	e, err := New()
	require.NoError(t, err)
	require.NoError(t, e.LoadPolicy(context.Background(), `
Decl score(Option, Value).
Decl bar(Value).
bar(950).
halt("Req", "below_bar", "T1") :- score("a", P), bar(T), :lt(P, T).
`))

	_, err = e.Explain(context.Background(), []string{`score("a", 0.9).`}, `halt("Req", Reason, Tier)`)
	require.Error(t, err, "a float operand must fail, not be coerced")
	assert.Contains(t, err.Error(), "is not a number")
}
