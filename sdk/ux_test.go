package sdk

import (
	"context"
	"testing"

	function "github.com/duynguyendang/manglekit/adapters/func"
	"github.com/duynguyendang/manglekit/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// WithHistory must compose with the default HybridMemory instead of
// discarding it (T-005).
func TestWithHistoryComposesWithHybridMemory(t *testing.T) {
	ctx := context.Background()

	store := &core.NopStore{}
	vec := core.NopVectorStore{}

	c, err := NewClient(ctx, WithMemory(NewHybridMemory(core.NopStore{}, vec, core.NopEmbedder{})), WithHistory(store))
	require.NoError(t, err)
	defer c.Shutdown(ctx)

	hm, ok := c.Memory().(*HybridMemory)
	require.True(t, ok, "memory should remain a HybridMemory")
	assert.Same(t, store, hm.History, "WithHistory should replace only the History component")
	assert.Equal(t, core.NopVectorStore{}, hm.Vectors, "vector store should be preserved")
}

// customMemory is a non-hybrid AgentMemory used to assert WithHistory fails
// loudly instead of discarding it.
type customMemory struct{ core.NopStore }

func (customMemory) Recall(_ context.Context, _ string) (string, error) { return "", nil }
func (customMemory) Memorize(_ context.Context, _, _ string) error      { return nil }
func (customMemory) Init(_ context.Context) error                       { return nil }

// WithHistory must fail loudly when a custom non-hybrid memory is already
// configured, rather than silently discarding it (T-005).
func TestWithHistoryRefusesToDiscardCustomMemory(t *testing.T) {
	_, err := NewClient(context.Background(), WithMemory(customMemory{}), WithHistory(&core.NopStore{}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "WithHistory")
	assert.Contains(t, err.Error(), "custom memory")
}

// TestWithMemory_InstallsMemory ensures WithMemory installs the memory.
func TestWithMemory_InstallsMemory(t *testing.T) {
	c, err := NewClient(context.Background(), WithMemory(nil))
	require.NoError(t, err)
	defer c.Shutdown(context.Background())
	assert.NotNil(t, c.Memory(), "nil must be ignored, keeping the default memory")
}

// WithPolicyPath must load the policy file at construction with typed
// errors (T-005/T-006).
func TestWithPolicyPath(t *testing.T) {
	ctx := context.Background()

	t.Run("loads policy", func(t *testing.T) {
		c, err := NewClient(ctx, WithPolicyPath("testdata/policy.dl"))
		require.NoError(t, err)
		defer c.Shutdown(ctx)
	})

	t.Run("missing file is a typed error", func(t *testing.T) {
		_, err := NewClient(ctx, WithPolicyPath("testdata/does-not-exist.dl"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read policy")
	})

	t.Run("policy path loads cleanly", func(t *testing.T) {
		c, err := NewClient(ctx, WithPolicyPath("testdata/policy.dl"))
		require.NoError(t, err)
		defer c.Shutdown(ctx)
	})
}

// RegisterSupervised must wrap and register in one call (T-006).
func TestRegisterSupervised(t *testing.T) {
	ctx := context.Background()
	c, err := NewClient(ctx, WithPolicyPath("testdata/policy.dl"))
	require.NoError(t, err)
	defer c.Shutdown(ctx)

	executed := false
	supervised := c.RegisterSupervised("noop_action", function.New("noop_action", func(_ context.Context, _ string) (string, error) {
		executed = true
		return "ok", nil
	}))
	require.NotNil(t, supervised)

	_, err = c.ExecuteByName(ctx, "noop_action", "payload")
	require.NoError(t, err)
	assert.True(t, executed, "registered supervised action should be executable by name")
}

// ExecuteByName must forward a caller-supplied envelope's explicit facts to
// the gate instead of dropping them (T-008).
func TestExecuteByNameForwardsEnvelopeFacts(t *testing.T) {
	ctx := context.Background()

	policy := `Decl flag(V).
halt("Req", "flag on", "T1") :- flag("on").`
	c, err := NewClient(ctx)
	require.NoError(t, err)
	defer c.Shutdown(ctx)
	require.NoError(t, c.LoadPolicy(ctx, policy))

	executed := false
	c.RegisterSupervised("flagged_action", function.New("flagged_action", func(_ context.Context, _ string) (string, error) {
		executed = true
		return "ok", nil
	}))

	env := core.NewEnvelope("payload")
	env.Facts = []string{`flag("on")`}

	_, err = c.ExecuteByName(ctx, "flagged_action", env)
	require.Error(t, err)
	assert.False(t, executed, "action must be blocked by the halt rule fed by the envelope fact")

	var viol *core.PolicyViolationError
	require.ErrorAs(t, err, &viol)
	assert.Equal(t, "flagged_action", viol.ActionName)
	assert.Contains(t, viol.MatchedRule, `halt("Req", "flag on", "T1")`)
	assert.Equal(t, "T1", viol.Tier)
	assert.True(t, core.IsPolicyViolationError(err))
}

// Without the fact the same action must proceed (control for the above).
func TestExecuteByNameProceedsWithoutEnvelopeFact(t *testing.T) {
	ctx := context.Background()

	policy := `Decl flag(V).
halt("Req", "flag on", "T1") :- flag("on").`
	c, err := NewClient(ctx)
	require.NoError(t, err)
	defer c.Shutdown(ctx)
	require.NoError(t, c.LoadPolicy(ctx, policy))

	executed := false
	c.RegisterSupervised("flagged_action", function.New("flagged_action", func(_ context.Context, _ string) (string, error) {
		executed = true
		return "ok", nil
	}))

	_, err = c.ExecuteByName(ctx, "flagged_action", "payload")
	require.NoError(t, err)
	assert.True(t, executed)
}
