package manglekit

import (
	"context"
	"testing"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/llm"
	"github.com/duynguyendang/manglekit/pipeline"
	"github.com/duynguyendang/manglekit/rerank"
	"github.com/duynguyendang/manglekit/retrieve"
	"github.com/firebase/genkit/go/ai"
)

// mockRetriever is a simple mock implementation of the retrieve.Retriever interface.
type mockRetriever struct{}

func (m *mockRetriever) Retrieve(_ context.Context, _ retrieve.Request) (retrieve.Result, error) {
	return retrieve.Result{Docs: []core.Doc{{Text: "mock"}}}, nil
}

// mockReranker is a simple mock implementation of the rerank.Reranker interface.
type mockReranker struct{}

func (m *mockReranker) Rerank(_ context.Context, _ rerank.Request) ([]rerank.ScoredDoc, error) {
	return []rerank.ScoredDoc{{Doc: core.Doc{Text: "mock"}}}, nil
}

// mockLLM is a simple mock implementation of the llm.Client interface.
type mockLLM struct{}

func (m *mockLLM) Complete(_ context.Context, _ llm.Request) (llm.Response, error) {
	return llm.Response{Text: "mock"}, nil
}

// mockRules is a simple mock implementation of the core.RuleSet interface.
type mockRules struct{}

func (m *mockRules) Evaluate(_ core.Stage, _ core.Query, _ *core.Answer) (core.RuleResult, error) {
	return core.RuleResult{Allowed: true}, nil
}

// mockEmbedder is a simple mock implementation of the ai.Embedder interface.
type mockEmbedder struct {
	ai.Embedder
}

// mockVectorStore is a simple mock implementation of the core.VectorStore interface.
type mockVectorStore struct{}

func (m *mockVectorStore) AddDocuments(context.Context, []core.Doc) error {
	return nil
}

func (m *mockVectorStore) Search(context.Context, string, []float32, int, map[string]any) ([]core.Doc, error) {
	return []core.Doc{{Text: "mock"}}, nil
}
func (m *mockVectorStore) Close(context.Context) error {
	return nil
}

type mockStateProvider struct {
	closed bool
}

func (m *mockStateProvider) Get(ctx context.Context, sessionID string) (interface{}, error) {
	return nil, nil
}
func (m *mockStateProvider) Set(ctx context.Context, sessionID string, state interface{}) error {
	return nil
}
func (m *mockStateProvider) Delete(ctx context.Context, sessionID string) error {
	return nil
}
func (m *mockStateProvider) Close(ctx context.Context) error {
	m.closed = true
	return nil
}

type mockOptions struct{}

type mockStateProviderOptions struct{}

func TestMain(m *testing.M) {
	RegisterRetriever("mock", func(opts any, deps FactoryDeps) (retrieve.Retriever, error) {
		return &mockRetriever{}, nil
	})
	RegisterReranker("mock", func(opts any, deps FactoryDeps) (rerank.Reranker, error) {
		return &mockReranker{}, nil
	})
	RegisterLLM("mock", func(opts any, deps FactoryDeps) (llm.Client, error) {
		return &mockLLM{}, nil
	})
	RegisterRules("mock", func(opts any) (core.RuleSet, error) {
		return &mockRules{}, nil
	})
	RegisterEmbedder("mock", func(opts any, deps FactoryDeps) (ai.Embedder, error) {
		return &mockEmbedder{}, nil
	})
	Register("mock-vs", func(opts any, deps FactoryDeps) (any, error) {
		return &mockVectorStore{}, nil
	})

	RegisterOptions("mock", (*mockOptions)(nil))
	RegisterOptions("mock-vs", (*core.LocalvecOptions)(nil))
	RegisterStateProvider("mock-sp", func(opts any, deps FactoryDeps) (core.StateProvider, error) {
		return &mockStateProvider{}, nil
	})
	RegisterOptions("mock-sp", (*mockStateProviderOptions)(nil))

	m.Run()
}

func TestBuilder(t *testing.T) {
	b := NewBuilder()
	if b == nil {
		t.Fatal("expected builder to be non-nil")
	}

	b.WithRetriever(&mockOptions{})
	b.WithReranker(&mockOptions{})
	b.WithLLM(&mockOptions{})
	b.WithRules(&mockOptions{})
	b.WithEmbedder(&mockOptions{})
	b.WithVectorStore(&core.LocalvecOptions{})

	if b.retrieverName != "mock" {
		t.Errorf("expected retriever name to be mock, got %s", b.retrieverName)
	}
	if b.rerankerName != "mock" {
		t.Errorf("expected reranker name to be mock, got %s", b.rerankerName)
	}
	if b.llmName != "mock" {
		t.Errorf("expected llm name to be mock, got %s", b.llmName)
	}
	if b.rulesName != "mock" {
		t.Errorf("expected rules name to be mock, got %s", b.rulesName)
	}
	if b.embedderName != "mock" {
		t.Errorf("expected embedder name to be mock, got %s", b.embedderName)
	}
	if b.vectorStoreName != "mock-vs" {
		t.Errorf("expected vector store name to be mock-vs, got %s", b.vectorStoreName)
	}
}

func TestBuild(t *testing.T) {
	b := NewBuilder()
	b.WithRetriever(&mockOptions{})
	b.WithLLM(&mockOptions{})
	b.WithEmbedder(&mockOptions{})
	b.WithVectorStore(&core.LocalvecOptions{})

	o, err := b.Build(context.Background())
	if err != nil {
		t.Fatalf("expected build to succeed, got %v", err)
	}
	if o == nil {
		t.Fatal("expected orchestrator to be non-nil")
	}
}

func TestBuildWithMockRerankerAndNoEmbedder(t *testing.T) {
	t.Parallel()
	builder := NewBuilder().
		WithRetriever(&mockOptions{}).
		WithReranker(&mockOptions{}).
		WithLLM(&mockOptions{})

	orch, err := builder.Build(context.Background())
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if orch == nil {
		t.Fatal("Build() returned nil orchestrator")
	}

	// Verify that a reranker was actually set.
	sandwich, ok := orch.(*pipeline.Sandwich)
	if !ok {
		t.Fatalf("Expected a Sandwich orchestrator, got %T", orch)
	}
	if sandwich.Reranker() == nil {
		t.Error("Orchestrator's reranker is nil, but a mock reranker should have been built")
	}

	if err := orch.Close(context.Background()); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestBuilderWithStateProvider(t *testing.T) {
	b := NewBuilder()
	b.WithStateProvider(&mockStateProviderOptions{})
	if b.stateProviderName != "mock-sp" {
		t.Errorf("expected state provider name to be mock-sp, got %s", b.stateProviderName)
	}
	b.WithRetriever(&mockOptions{})
	b.WithLLM(&mockOptions{})
	o, err := b.Build(context.Background())
	if err != nil {
		t.Fatalf("expected build to succeed, got %v", err)
	}
	if o == nil {
		t.Fatal("expected orchestrator to be non-nil")
	}
	// a bit of a hack to check if the closer was called
	p := o.(*pipeline.Sandwich)
	sp := p.StateProvider().(*mockStateProvider)
	if err := o.Close(context.Background()); err != nil {
		t.Errorf("Close() error = %v", err)
	}
	if !sp.closed {
		t.Error("expected state provider to be closed")
	}
}
