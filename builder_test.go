package manglekit

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
	"github.com/duynguyendang/manglekit/retrieve"
	"github.com/firebase/genkit/go/ai"
)

// mockEmbedder is a dummy embedder for testing.
type mockEmbedder struct {
	ai.Embedder
}

func mockEmbedderFactory(ctx context.Context, deps diapi.EmbedderDeps, cfg any) (ai.Embedder, error) {
	return &mockEmbedder{}, nil
}

// retrieverFactoryThatNeedsEmbedder is a test factory that requires an embedder.
func retrieverFactoryThatNeedsEmbedder(ctx context.Context, deps diapi.RetrieverDeps, cfg any) (retrieve.Retriever, error) {
	if deps.Embedder == nil {
		return nil, fmt.Errorf("test retriever requires an embedder")
	}
	return &retrieve.NoopRetriever{}, nil
}

// mockOrchestrator is a dummy orchestrator for testing.
type mockOrchestrator struct {
	core.Orchestrator
}

func mockOrchestratorFactory(opts core.Options) (core.Orchestrator, error) {
	return &mockOrchestrator{}, nil
}

func TestBuilder_DependencyInjection_Failure(t *testing.T) {
	t.Run("should fail when required dependency is missing", func(t *testing.T) {
		r := NewRegistry()
		r.RegisterRetriever("needs-embedder", retrieverFactoryThatNeedsEmbedder)
		r.RegisterOrchestrator("sandwich", mockOrchestratorFactory)

		type TestRetrieverOptions struct{}
		r.RegisterOptions("needs-embedder", (*TestRetrieverOptions)(nil))

		builder := NewBuilder(r)

		// Configure the retriever that needs an embedder, but do NOT provide an embedder.
		builder.WithRetriever(&TestRetrieverOptions{})

		_, err := builder.Build(context.Background())
		if err == nil {
			t.Fatal("expected builder.Build() to fail, but it succeeded")
		}

		expectedErrorMsg := "test retriever requires an embedder"
		if !strings.Contains(err.Error(), expectedErrorMsg) {
			t.Errorf("expected error message to contain %q, but got: %v", expectedErrorMsg, err)
		}
	})
}

func TestBuilder_DependencyInjection_Success(t *testing.T) {
	t.Run("should succeed when required dependency is present", func(t *testing.T) {
		r := NewRegistry()
		r.RegisterRetriever("needs-embedder", retrieverFactoryThatNeedsEmbedder)
		r.RegisterEmbedder("mock-embedder", mockEmbedderFactory)
		r.RegisterOrchestrator("sandwich", mockOrchestratorFactory)

		type TestRetrieverOptions struct{}
		r.RegisterOptions("needs-embedder", (*TestRetrieverOptions)(nil))

		type MockEmbedderOptions struct{}
		r.RegisterOptions("mock-embedder", (*MockEmbedderOptions)(nil))

		builder := NewBuilder(r)

		// Configure the retriever AND provide the embedder dependency.
		builder.WithRetriever(&TestRetrieverOptions{})
		builder.WithEmbedder(&MockEmbedderOptions{})

		_, err := builder.Build(context.Background())
		if err != nil {
			t.Fatalf("expected builder.Build() to succeed, but it failed: %v", err)
		}
	})
}