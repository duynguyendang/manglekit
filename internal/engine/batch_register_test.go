package engine

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/duynguyendang/manglekit/core"
)

func TestRegisterActions_TriggersExactlyOneEvaluation(t *testing.T) {
	pe, err := New()
	require.NoError(t, err)

	// Warm the runtime (stdlib load) and reset the counter.
	pe.runtime.mu.Lock()
	pe.runtime.evalCount = 0
	pe.runtime.mu.Unlock()

	metas := make([]core.ActionMetadata, 0, 10)
	for i := 0; i < 10; i++ {
		metas = append(metas, core.ActionMetadata{
			Name:       "act_" + string(rune('a'+i)),
			InputType:  "string",
			OutputType: "string",
		})
	}
	require.NoError(t, pe.RegisterActions(metas))

	pe.runtime.mu.Lock()
	evalsWithBatch := pe.runtime.evalCount
	pe.runtime.mu.Unlock()
	assert.Equal(t, 1, evalsWithBatch, "batch registration must trigger exactly one re-evaluation")

	// Control: the single-item API triggers one evaluation per action.
	pe.runtime.mu.Lock()
	pe.runtime.evalCount = 0
	pe.runtime.mu.Unlock()

	for _, m := range metas {
		require.NoError(t, pe.RegisterAction(m))
	}

	pe.runtime.mu.Lock()
	evalsWithSingle := pe.runtime.evalCount
	pe.runtime.mu.Unlock()
	assert.Equal(t, len(metas), evalsWithSingle)

	// The registered actions are queryable in both cases.
	ok, err := pe.ExecuteQuery(context.Background(), nil, `action("act_a")`)
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestRegisterActions_Empty(t *testing.T) {
	pe, err := New()
	require.NoError(t, err)
	require.NoError(t, pe.RegisterActions(nil))
}
