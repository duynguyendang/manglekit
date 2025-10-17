package inmemory

import (
	"context"
	"testing"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core/diapi"
	"github.com/duynguyendang/manglekit/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemoryStateProvider(t *testing.T) {
	ctx := context.Background()

	t.Run("New", func(t *testing.T) {
		p, err := New(state.InMemoryOptions{})
		require.NoError(t, err)
		assert.NotNil(t, p)
	})

	t.Run("Set and Get", func(t *testing.T) {
		p, _ := New(state.InMemoryOptions{})
		sessionID := "session-1"
		stateData := "test-state"

		err := p.Set(ctx, sessionID, stateData)
		require.NoError(t, err)

		retrievedState, err := p.Get(ctx, sessionID)
		require.NoError(t, err)
		assert.Equal(t, stateData, retrievedState)
	})

	t.Run("Get non-existent", func(t *testing.T) {
		p, _ := New(state.InMemoryOptions{})
		retrievedState, err := p.Get(ctx, "non-existent-session")
		require.NoError(t, err)
		assert.Nil(t, retrievedState)
	})

	t.Run("Delete", func(t *testing.T) {
		p, _ := New(state.InMemoryOptions{})
		sessionID := "session-to-delete"
		stateData := "data-to-delete"

		p.Set(ctx, sessionID, stateData)

		err := p.Delete(ctx, sessionID)
		require.NoError(t, err)

		retrievedState, err := p.Get(ctx, sessionID)
		require.NoError(t, err)
		assert.Nil(t, retrievedState)
	})

	t.Run("Delete non-existent", func(t *testing.T) {
		p, _ := New(state.InMemoryOptions{})
		err := p.Delete(ctx, "non-existent-session")
		require.NoError(t, err)
	})

	t.Run("Close", func(t *testing.T) {
		p, _ := New(state.InMemoryOptions{})
		err := p.Close(ctx)
		require.NoError(t, err)
	})
}

func TestRegister(t *testing.T) {
	r := manglekit.NewRegistry()
	Register(r)

	t.Run("provider is registered", func(t *testing.T) {
		factory, ok := r.StateProviders["inmemory"]
		require.True(t, ok)
		assert.NotNil(t, factory)

		// Test the factory itself
		provider, err := factory(context.Background(), diapi.StateProviderDeps{}, &state.InMemoryOptions{})
		require.NoError(t, err)
		assert.IsType(t, &Provider{}, provider)
	})

	t.Run("options are registered", func(t *testing.T) {
		optsType, ok := r.NameToOptionsType("inmemory")
		require.True(t, ok)
		assert.Equal(t, "InMemoryOptions", optsType.Elem().Name())
	})

	t.Run("factory handles nil config", func(t *testing.T) {
		factory, _ := r.StateProviders["inmemory"]
		provider, err := factory(context.Background(), diapi.StateProviderDeps{}, nil)
		require.NoError(t, err)
		assert.IsType(t, &Provider{}, provider)
	})

	t.Run("factory handles wrong config type", func(t *testing.T) {
		factory, _ := r.StateProviders["inmemory"]
		type WrongOptions struct{}
		_, err := factory(context.Background(), diapi.StateProviderDeps{}, &WrongOptions{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid options type")
	})
}