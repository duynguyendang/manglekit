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
