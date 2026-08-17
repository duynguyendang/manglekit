package engine

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"codeberg.org/TauCeti/mangle-go/ast"
	"codeberg.org/TauCeti/mangle-go/parse"
	"github.com/duynguyendang/manglekit/core"
)

func TestIDBCache_ReusesEvaluationForIdenticalFacts(t *testing.T) {
	rt := NewMangleRuntime()
	ctx := context.Background()
	require.NoError(t, rt.AddPolicy(ctx, "Decl blocked(A0).\nhalt(\"Req\",\"no\") :- blocked(\"yes\")."))

	facts := []string{`blocked("yes").`}
	atoms := mustParseAtoms(t, facts)

	rt.mu.RLock()
	before := rt.evalCount
	rt.mu.RUnlock()

	ok, err := rt.ExecuteQuery(ctx, atoms, `halt("Req", "no")`)
	require.NoError(t, err)
	assert.True(t, ok)

	// Repeated queries with identical request facts must reuse the cached
	// evaluated store instead of re-running stratified evaluation.
	for i := 0; i < 5; i++ {
		ok, err := rt.ExecuteQuery(ctx, atoms, `halt("Req", "no")`)
		require.NoError(t, err)
		assert.True(t, ok)
	}

	rt.mu.RLock()
	after := rt.evalCount
	rt.mu.RUnlock()
	assert.Equal(t, before+1, after, "identical fact sets should trigger exactly one evaluation")
}

func TestIDBCache_InvalidationOnChangeFacts(t *testing.T) {
	rt := NewMangleRuntime()
	ctx := context.Background()
	require.NoError(t, rt.AddPolicy(ctx, "Decl blocked(A0).\nhalt(\"Req\",\"no\") :- blocked(\"yes\")."))

	ok, err := rt.ExecuteQuery(ctx, nil, `halt("Req", "no")`)
	require.NoError(t, err)
	assert.False(t, ok, "no blocked fact yet: gate must pass")

	// Adding a base fact must invalidate the cache and change the result.
	require.NoError(t, rt.LoadFacts(ctx, []string{`blocked("yes").`}))

	ok, err = rt.ExecuteQuery(ctx, nil, `halt("Req", "no")`)
	require.NoError(t, err)
	assert.True(t, ok, "after loading blocked(\"yes\") the gate must halt")
}

func TestIDBCache_InvalidationOnChangePolicy(t *testing.T) {
	rt := NewMangleRuntime()
	ctx := context.Background()
	require.NoError(t, rt.AddPolicy(ctx, `hold_true("a").`))

	ok, err := rt.ExecuteQuery(ctx, nil, `halt_new("x")`)
	require.NoError(t, err)
	assert.False(t, ok)

	// Adding a policy that derives the queried predicate must invalidate.
	require.NoError(t, rt.AddPolicy(ctx, `halt_new("x") :- hold_true("a").`))

	ok, err = rt.ExecuteQuery(ctx, nil, `halt_new("x")`)
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestIDBCache_GateCheckReflectsBaseFactChange(t *testing.T) {
	pe, err := New()
	require.NoError(t, err)
	ctx := context.Background()
	require.NoError(t, pe.LoadPolicy(ctx, "Decl secret_present(A0).\nhalt(\"Req\",\"secret\") :- secret_present(\"yes\")."))

	env := core.Envelope{Metadata: map[string]any{"user": "alice"}}
	require.NoError(t, pe.Assess(ctx, core.ActionMetadata{Name: "read"}, env))

	// Prime the cache with a deny-producing fact set.
	require.NoError(t, pe.LoadFacts(ctx, []string{`secret_present("yes").`}))
	err = pe.Assess(ctx, core.ActionMetadata{Name: "read"}, env)
	require.Error(t, err, "gate must halt once secret_present is a base fact")

	// A fresh runtime (simulating fact removal) passes again — the cached
	// result from the old fact set must not stick after invalidation.
	pe2, err := New()
	require.NoError(t, err)
	require.NoError(t, pe2.LoadPolicy(ctx, "Decl secret_present(A0).\nhalt(\"Req\",\"secret\") :- secret_present(\"yes\")."))
	require.NoError(t, pe2.Assess(ctx, core.ActionMetadata{Name: "read"}, env))
}

func mustParseAtoms(t *testing.T, facts []string) []ast.Atom {
	t.Helper()
	var atoms []ast.Atom
	for _, f := range facts {
		a, err := parse.Atom(f)
		require.NoError(t, err)
		atoms = append(atoms, a)
	}
	return atoms
}

// benchGateEngine builds a PolicyEngine with a halt policy and a 1k-fact
// base store for the gate-check benchmark.
func benchGateEngine(b *testing.B) *PolicyEngine {
	pe, err := New()
	if err != nil {
		b.Fatal(err)
	}
	if err := pe.LoadPolicy(context.Background(), "Decl blocked(A0).\nhalt(\"Req\",\"blocked\") :- blocked(\"yes\")."); err != nil {
		b.Fatal(err)
	}
	facts := make([]string, 0, 1000)
	for i := 0; i < 1000; i++ {
		facts = append(facts, fmt.Sprintf(`item("key%d", "value%d").`, i, i))
	}
	if err := pe.LoadFacts(context.Background(), facts); err != nil {
		b.Fatal(err)
	}
	return pe
}

// BenchmarkGateCheck1kFacts measures a supervised pre-check (Assess) against
// a 1k-fact store with the derived-fact (IDB) cache enabled.
func BenchmarkGateCheck1kFacts(b *testing.B) {
	pe := benchGateEngine(b)
	ctx := context.Background()
	meta := core.ActionMetadata{Name: "bench_action"}
	env := core.Envelope{Metadata: map[string]any{"user": "alice"}}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := pe.Assess(ctx, meta, env); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkGateCheck1kFacts_NoCache is the control: same workload with the
// IDB cache disabled (the pre-T-001 behavior).
func BenchmarkGateCheck1kFacts_NoCache(b *testing.B) {
	pe := benchGateEngine(b)
	pe.runtime.disableIDBCache = true
	ctx := context.Background()
	meta := core.ActionMetadata{Name: "bench_action"}
	env := core.Envelope{Metadata: map[string]any{"user": "alice"}}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := pe.Assess(ctx, meta, env); err != nil {
			b.Fatal(err)
		}
	}
}
