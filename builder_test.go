package manglekit_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
	"github.com/duynguyendang/manglekit/internal/providers/orchestrators"
	"github.com/duynguyendang/manglekit/llm"
	llmprovider "github.com/duynguyendang/manglekit/internal/providers/llm"
	"github.com/duynguyendang/manglekit/pipeline/declarative"
	"github.com/duynguyendang/manglekit/retrieve"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/stretchr/testify/assert"
)

// Compile-time check to ensure DeclarativeOrchestrator implements the Orchestrator interface.
var _ core.Orchestrator = (*declarative.DeclarativeOrchestrator)(nil)

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
		r := manglekit.NewRegistry()
		r.RegisterRetriever("needs-embedder", retrieverFactoryThatNeedsEmbedder)
		r.RegisterOrchestrator("sandwich", mockOrchestratorFactory)

		type TestRetrieverOptions struct{}
		r.RegisterOptions("needs-embedder", (*TestRetrieverOptions)(nil))

		builder := manglekit.NewBuilder(r)

		// Configure the retriever that needs an embedder, but do NOT provide an embedder.
		builder.WithRetriever(&TestRetrieverOptions{})

		_, _, err := builder.Build(context.Background())
		if err == nil {
			t.Fatal("expected builder.Build() to fail, but it succeeded")
		}

		expectedErrorMsg := "test retriever requires an embedder"
		if !strings.Contains(err.Error(), expectedErrorMsg) {
			t.Errorf("expected error message to contain %q, but got: %v", expectedErrorMsg, err)
		}
	})
}

func TestBuilder_OrchestratorSelection(t *testing.T) {
	// mock orchestrators for testing selection
	type mockSandwich struct{ core.Orchestrator }
	sandwichFactory := func(opts core.Options) (core.Orchestrator, error) { return &mockSandwich{}, nil }

	type mockDeclarative struct{ core.Orchestrator }
	declarativeFactory := func(opts core.Options) (core.Orchestrator, error) { return &mockDeclarative{}, nil }

	r := manglekit.NewRegistry()
	r.RegisterOrchestrator("sandwich", sandwichFactory)
	r.RegisterOrchestrator("declarative", declarativeFactory)

	t.Run("should use default orchestrator when none is specified", func(t *testing.T) {
		builder := manglekit.NewBuilder(r)
		orch, _, err := builder.Build(context.Background())
		assert.NoError(t, err)
		assert.IsType(t, &mockSandwich{}, orch, "expected default orchestrator to be mockSandwich")
	})

	t.Run("should use the specified orchestrator", func(t *testing.T) {
		builder := manglekit.NewBuilder(r).WithOrchestrator("declarative")
		orch, _, err := builder.Build(context.Background())
		assert.NoError(t, err)
		assert.IsType(t, &mockDeclarative{}, orch, "expected selected orchestrator to be mockDeclarative")
	})

	t.Run("should return an error for an unknown orchestrator", func(t *testing.T) {
		builder := manglekit.NewBuilder(r).WithOrchestrator("non-existent")
		_, _, err := builder.Build(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), `unknown orchestrator "non-existent"`)
		assert.Contains(t, err.Error(), "known orchestrators:")
		assert.Contains(t, err.Error(), "sandwich")
		assert.Contains(t, err.Error(), "declarative")
	})
}

func TestBuilder_WithGenkit_OpenAI(t *testing.T) {
	// This test requires an OpenAI API key to be set in the environment.
	// In a CI environment, this would be skipped if the key is not present.
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("skipping test: OPENAI_API_KEY not set")
	}

	r := manglekit.NewRegistry()
	llmprovider.Register(r) // Register all LLM providers
	orchestrators.Register(r)

	// A bare genkit instance is enough now. The provider handles the plugin logic.
	g := genkit.Init(context.Background())

	builder := manglekit.NewBuilder(r)
	builder.WithGenkit(g)
	builder.WithLLM(&llm.OpenAIOptions{Model: "gpt-4o-mini", APIKey: apiKey})

	_, _, err := builder.Build(context.Background())
	assert.NoError(t, err, "Builder.Build() should succeed with a Genkit-configured OpenAI provider")
}

func TestBuilder_DependencyInjection_Success(t *testing.T) {
	t.Run("should succeed when required dependency is present", func(t *testing.T) {
		r := manglekit.NewRegistry()
		r.RegisterRetriever("needs-embedder", retrieverFactoryThatNeedsEmbedder)
		r.RegisterEmbedder("mock-embedder", mockEmbedderFactory)
		r.RegisterOrchestrator("sandwich", mockOrchestratorFactory)

		type TestRetrieverOptions struct{}
		r.RegisterOptions("needs-embedder", (*TestRetrieverOptions)(nil))

		type MockEmbedderOptions struct{}
		r.RegisterOptions("mock-embedder", (*MockEmbedderOptions)(nil))

		builder := manglekit.NewBuilder(r)

		// Configure the retriever AND provide the embedder dependency.
		builder.WithRetriever(&TestRetrieverOptions{})
		builder.WithEmbedder(&MockEmbedderOptions{})

		_, _, err := builder.Build(context.Background())
		if err != nil {
			t.Fatalf("expected builder.Build() to succeed, but it failed: %v", err)
		}
	})
}
