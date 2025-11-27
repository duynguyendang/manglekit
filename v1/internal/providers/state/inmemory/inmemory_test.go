package inmemory_test

import (
	"context"
	"testing"
	"time"

	"github.com/duynguyendang/manglekit/v1/core/diapi"
	"github.com/duynguyendang/manglekit/v1/internal/providers/state/inmemory"
	"github.com/stretchr/testify/require"
)

func TestInMemory_State_PutGet_Close(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	p, err := inmemory.New(inmemory.Options{ContextWindow: 3}, diapi.StateProviderDeps{})
	require.NoError(t, err)

	// Put and Get "turn-1"
	err = p.Set(ctx, "sess-1", "turn-1")
	require.NoError(t, err)

	val, err := p.Get(ctx, "sess-1")
	require.NoError(t, err)
	require.Equal(t, "turn-1", val)

	// Overwrite with "turn-2" and Get again
	err = p.Set(ctx, "sess-1", "turn-2")
	require.NoError(t, err)

	val, err = p.Get(ctx, "sess-1")
	require.NoError(t, err)
	require.Equal(t, "turn-2", val)

	// Verify Close
	err = p.Close(ctx)
	require.NoError(t, err)

	// Verify that Get and Set fail after Close
	_, err = p.Get(ctx, "sess-1")
	require.ErrorIs(t, err, inmemory.ErrProviderClosed)

	err = p.Set(ctx, "sess-1", "turn-3")
	require.ErrorIs(t, err, inmemory.ErrProviderClosed)
}
