package hybrid

import (
	"context"
	"errors"
	"testing"

	"github.com/duynguyendang/manglekit/v1"
	"github.com/duynguyendang/manglekit/v1/core"
	"github.com/duynguyendang/manglekit/v1/core/diapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRetriever is a mock implementation of core.Retriever for testing.
type mockRetriever struct {
	RetrieveFunc func(ctx context.Context, req core.RetrieveRequest) (core.RetrieveResult, error)
}

func (m *mockRetriever) Retrieve(ctx context.Context, req core.RetrieveRequest) (core.RetrieveResult, error) {
	if m.RetrieveFunc != nil {
		return m.RetrieveFunc(ctx, req)
	}
	return core.RetrieveResult{}, errors.New("RetrieveFunc not implemented")
}

func TestNew(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		retrievers := []core.Retriever{&mockRetriever{}}
		retriever, err := New(retrievers, 60.0, diapi.RetrieverDeps{})
		require.NoError(t, err)
		assert.NotNil(t, retriever)
	})

	t.Run("no_retrievers", func(t *testing.T) {
		_, err := New([]core.Retriever{}, 60.0, diapi.RetrieverDeps{})
		assert.Error(t, err)
	})
}

func TestHybrid_Retrieve_RRF(t *testing.T) {
	ctx := context.Background()
	retriever1 := &mockRetriever{
		RetrieveFunc: func(ctx context.Context, req core.RetrieveRequest) (core.RetrieveResult, error) {
			return core.RetrieveResult{
				Docs: []core.Doc{{ID: "A"}, {ID: "B"}}, // A is rank 0, B is rank 1
			}, nil
		},
	}
	retriever2 := &mockRetriever{
		RetrieveFunc: func(ctx context.Context, req core.RetrieveRequest) (core.RetrieveResult, error) {
			return core.RetrieveResult{
				Docs: []core.Doc{{ID: "B"}, {ID: "A"}}, // B is rank 0, A is rank 1
			}, nil
		},
	}

	hybrid, err := New([]core.Retriever{retriever1, retriever2}, 60.0, diapi.RetrieverDeps{})
	require.NoError(t, err)

	result, err := hybrid.Retrieve(ctx, core.RetrieveRequest{TopK: 2})
	require.NoError(t, err)

	// With RRF, both documents get the same score (1/(60+0) + 1/(60+1)),
	// so the final order is not guaranteed. We only check the contents.
	assert.Len(t, result.Docs, 2)
	assert.ElementsMatch(t, []string{"A", "B"}, []string{result.Docs[0].ID, result.Docs[1].ID})
}

func TestHybrid_Retrieve_SingleRetriever(t *testing.T) {
	ctx := context.Background()
	mockResult := core.RetrieveResult{Docs: []core.Doc{{ID: "A"}}}
	retriever := &mockRetriever{
		RetrieveFunc: func(ctx context.Context, req core.RetrieveRequest) (core.RetrieveResult, error) {
			return mockResult, nil
		},
	}

	hybrid, err := New([]core.Retriever{retriever}, 60.0, diapi.RetrieverDeps{})
	require.NoError(t, err)

	result, err := hybrid.Retrieve(ctx, core.RetrieveRequest{TopK: 1})
	require.NoError(t, err)
	assert.Equal(t, mockResult, result)
}

// mockBuilder is a mock implementation of diapi.Builder for testing.
type mockBuilder struct {
	retrievers map[string]core.Retriever
}

func TestHybrid_Factory(t *testing.T) {
	r := manglekit.NewRegistry()
	Register(r)

	factory, err := r.Get(core.KindRetriever, "hybrid")
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		opts := &HybridOptions{
			Retrievers: []string{"r1", "r2"},
		}
		deps := diapi.RetrieverDeps{
			SubRetrievers: map[string]core.Retriever{
				"r1": &mockRetriever{},
				"r2": &mockRetriever{},
			},
		}
		retriever, err := factory.Build(context.Background(), deps, opts)
		require.NoError(t, err)
		assert.NotNil(t, retriever)
	})

	t.Run("missing_retriever", func(t *testing.T) {
		opts := &HybridOptions{
			Retrievers: []string{"r1", "r3"}, // r3 does not exist
		}
		deps := diapi.RetrieverDeps{
			SubRetrievers: map[string]core.Retriever{
				"r1": &mockRetriever{},
			},
		}
		_, err := factory.Build(context.Background(), deps, opts)
		assert.Error(t, err)
	})
}
