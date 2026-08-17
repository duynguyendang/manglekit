package engine

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const reloadPolicyV1 = `
Decl flagged(X).
risky(X) :- flagged(X).
halt(Req, "denied by v1", "T1") :- risky(Req).
`

const reloadPolicyV2 = `
Decl flagged(X).
halt(Req, "denied by v2", "T2") :- flagged(Req).
`

func TestReloadFromSource_AtomicAndFailSafe(t *testing.T) {
	e, err := New()
	require.NoError(t, err)

	require.NoError(t, e.LoadFacts(context.Background(), []string{`flagged("Req").`}))
	require.NoError(t, e.runtime.AddPolicy(context.Background(), reloadPolicyV1))

	ctx := context.Background()

	// v1 active.
	expl, err := e.Explain(ctx, nil, `halt("Req", Reason, Tier)`)
	require.NoError(t, err)
	require.True(t, expl.Outcome)
	assert.Contains(t, expl.Derivations[0].Fact, "denied by v1")

	// Failed reload (parse error) keeps v1.
	err = e.ReloadPolicySource(ctx, "halt(((")
	require.Error(t, err)
	expl, err = e.Explain(ctx, nil, `halt("Req", Reason, Tier)`)
	require.NoError(t, err)
	require.True(t, expl.Outcome)
	assert.Contains(t, expl.Derivations[0].Fact, "denied by v1")

	// Successful reload to v2: rules replaced, base facts preserved.
	require.NoError(t, e.ReloadPolicySource(ctx, reloadPolicyV2))
	expl, err = e.Explain(ctx, nil, `halt("Req", Reason, Tier)`)
	require.NoError(t, err)
	require.True(t, expl.Outcome)
	assert.Contains(t, expl.Derivations[0].Fact, "denied by v2")

	// v1's intermediate predicate must be gone.
	ok, err := e.ExecuteQuery(ctx, nil, `risky("Req")`)
	require.NoError(t, err)
	assert.False(t, ok, "rules from the old policy must not survive a reload")
}

func TestReloadFromSource_ConcurrentQueriesNeverSeePartialPolicy(t *testing.T) {
	e, err := New()
	require.NoError(t, err)
	require.NoError(t, e.LoadFacts(context.Background(), []string{`flagged("Req").`}))
	require.NoError(t, e.runtime.AddPolicy(context.Background(), reloadPolicyV1))

	ctx := context.Background()
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 200; i++ {
			found := false
			qErr := e.runtime.QueryWithSolutions(ctx, nil, `halt("Req", Reason, Tier)`, func(map[string]any) error {
				found = true
				return nil
			})
			assert.NoError(t, qErr)
			assert.True(t, found, "query during reload must always find a halt fact (old or new policy)")
		}
	}()

	for i := 0; i < 20; i++ {
		require.NoError(t, e.ReloadPolicySource(ctx, reloadPolicyV2))
		require.NoError(t, e.ReloadPolicySource(ctx, reloadPolicyV1))
	}
	<-done
}

// Regression: explicitly loaded facts whose predicate is IDB in the OLD
// program (triple/3 is IDB because std.dl derives it from quad/4) must
// survive a LoadFromSource reload. Dropping them wiped knowledge-graph
// facts the replacement policy needed (hybrid_rag Feature 4).
func TestReloadFromSource_PreservesExplicitFactsOfOldIDBPredicates(t *testing.T) {
	e, err := New()
	require.NoError(t, err)
	ctx := context.Background()

	// Load a first policy so triple/3 is "IDB" in the old program.
	require.NoError(t, e.runtime.LoadFromSource(ctx, `Decl doc(X).
allow(X) :- triple(X, "visible", "true").`))

	// Explicitly load triple facts while the first policy is active.
	// The "bob" fact also makes the OLD policy derive allow("bob") —
	// a derivation that must NOT survive the reload below.
	require.NoError(t, e.LoadFacts(ctx, []string{
		`triple("bob", "visible", "true")`,
		`triple("alice", "member_of", "team_alpha")`,
		`triple("team_alpha", "contributor_to", "repo_backend")`,
		`triple("repo_backend", "contains_module", "module_auth")`,
	}))

	// Replace the policy; the new rules join over the same triples.
	require.NoError(t, e.runtime.LoadFromSource(ctx, `Decl doc(X).
can_access(User, Module) :-
    triple(User, "member_of", Team),
    triple(Team, "contributor_to", Repo),
    triple(Repo, "contains_module", Module).`))

	found := false
	require.NoError(t, e.runtime.QueryWithSolutions(ctx, nil, `can_access("alice", "module_auth")`, func(map[string]any) error {
		found = true
		return nil
	}))
	assert.True(t, found, "explicitly loaded triple facts must survive a policy reload")

	// Stale derivations of the OLD policy's rules are still dropped.
	found = false
	require.NoError(t, e.runtime.QueryWithSolutions(ctx, nil, `allow("alice")`, func(map[string]any) error {
		found = true
		return nil
	}))
	assert.False(t, found, "old-policy derivations must not survive a reload")
}
