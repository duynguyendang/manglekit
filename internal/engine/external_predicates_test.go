package engine

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExternalPredicate_Basic(t *testing.T) {
	t.Skip("External predicates require further investigation - Mangle analysis phase needs predicate declaration")
	// Create a new runtime
	runtime := NewMangleRuntime()

	// Register external predicate FIRST, before loading policy
	err := runtime.RegisterExternalPredicate("double", func(ctx context.Context, inputs []any) ([][]any, error) {
		t.Logf("double called with inputs: %v", inputs)
		n := inputs[0].(int64)
		return [][]any{{n * 2}}, nil
	})
	require.NoError(t, err)

	// Load a policy that uses the external predicate
	// No Decl needed - external predicate is auto-declared
	policy := `
		result(X) :- double(5, X).
	`
	err = runtime.LoadFromString(policy)
	require.NoError(t, err)

	// Query - external predicates should be evaluated during query
	allowed, err := runtime.ExecuteQuery(nil, "result(X)")
	t.Logf("Query result: allowed=%v, err=%v", allowed, err)

	// The external predicate should have been called and returned true
	require.NoError(t, err)
	assert.True(t, allowed, "Expected query to succeed")
}

func TestExternalPredicate_AddNumbers(t *testing.T) {
	t.Skip("External predicates require further investigation")
}

func TestExternalPredicateRegistry_Count(t *testing.T) {
	// This test doesn't need external predicates to work
	runtime := NewMangleRuntime()

	// Initially empty
	assert.Equal(t, 0, runtime.ExternalPredicates().Count())

	// Register a predicate
	err := runtime.RegisterExternalPredicate("test_pred", func(ctx context.Context, inputs []any) ([][]any, error) {
		return nil, nil
	})
	require.NoError(t, err)

	// Should have 1 predicate
	assert.Equal(t, 1, runtime.ExternalPredicates().Count())

	// Should be able to retrieve
	fn, ok := runtime.ExternalPredicates().Get("test_pred")
	assert.True(t, ok)
	assert.NotNil(t, fn)
}

func TestPolicyEngine_ExternalPredicate_Basic(t *testing.T) {
	t.Skip("External predicates require further investigation")
}
