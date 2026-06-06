package engine

import (
	"context"
	"testing"

	"codeberg.org/TauCeti/mangle-go/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExternalPredicate_Basic(t *testing.T) {
	runtime := NewMangleRuntime()

	err := runtime.RegisterExternalPredicate("double", func(ctx context.Context, inputs []any) ([][]any, error) {
		n := inputs[0].(int64)
		return [][]any{{n * 2}}, nil
	})
	require.NoError(t, err)

	policy := `result(X) :- double(5, X).`
	err = runtime.LoadFromString(policy)
	require.NoError(t, err)

	// External predicate adds facts to base store during evaluation.
	// Verify the derived fact exists in the store.
	ok, err := runtime.ExecuteQuery(context.Background(), nil, "result(10)")
	require.NoError(t, err)
	assert.True(t, ok, "result(10) should exist after double(5) returns 10")
}

func TestExternalPredicate_Chained(t *testing.T) {
	runtime := NewMangleRuntime()

	err := runtime.RegisterExternalPredicate("add_one", func(ctx context.Context, inputs []any) ([][]any, error) {
		n := inputs[0].(int64)
		return [][]any{{n + 1}}, nil
	})
	require.NoError(t, err)

	policy := `result(X) :- add_one(10, X).`
	err = runtime.LoadFromString(policy)
	require.NoError(t, err)

	ok, err := runtime.ExecuteQuery(context.Background(), nil, "result(11)")
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestExternalPredicate_InBodyChain(t *testing.T) {
	runtime := NewMangleRuntime()

	err := runtime.RegisterExternalPredicate("double", func(ctx context.Context, inputs []any) ([][]any, error) {
		n := inputs[0].(int64)
		return [][]any{{n * 2}}, nil
	})
	require.NoError(t, err)

	policy := `
		intermediate(X) :- double(5, X).
		final(Y) :- intermediate(X), Y = X.
	`
	err = runtime.LoadFromString(policy)
	require.NoError(t, err)

	ok, err := runtime.ExecuteQuery(context.Background(), nil, "intermediate(10)")
	require.NoError(t, err)
	assert.True(t, ok, "intermediate(10) should be derived")
}

func TestExternalPredicate_BaseStoreDirect(t *testing.T) {
	runtime := NewMangleRuntime()

	called := false
	err := runtime.RegisterExternalPredicate("triple", func(ctx context.Context, inputs []any) ([][]any, error) {
		called = true
		n := inputs[0].(int64)
		return [][]any{{n * 3}}, nil
	})
	require.NoError(t, err)

	policy := `result(X) :- triple(3, X).`
	err = runtime.LoadFromString(policy)
	require.NoError(t, err)
	assert.True(t, called, "external predicate should be called during evaluation")

	// Check base store directly
	var found bool
	runtime.baseFactStore.GetFacts(
		ast.Atom{Predicate: ast.PredicateSym{Symbol: "result", Arity: 1}},
		func(fact ast.Atom) error {
			if len(fact.Args) > 0 {
				if c, ok := fact.Args[0].(ast.Constant); ok && c.NumValue == 9 {
					found = true
				}
			}
			return nil
		},
	)
	assert.True(t, found, "result(9) should be in base store (triple(3) = 9)")
}

func TestExternalPredicateRegistry_Count(t *testing.T) {
	runtime := NewMangleRuntime()
	assert.Equal(t, 0, runtime.ExternalPredicates().Count())

	err := runtime.RegisterExternalPredicate("test_pred", func(ctx context.Context, inputs []any) ([][]any, error) {
		return nil, nil
	})
	require.NoError(t, err)
	assert.Equal(t, 1, runtime.ExternalPredicates().Count())

	fn, ok := runtime.ExternalPredicates().Get("test_pred")
	assert.True(t, ok)
	assert.NotNil(t, fn)
}
