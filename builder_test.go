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
	"github.com/firebase/genkit/go/core/api"
)

// mockRetriever is a simple mock implementation of the retrieve.Retriever interface.
type mockRetriever struct{}

func (m *mockRetriever) Retrieve(_ context.Context, _ retrieve.Request) (retrieve.Result, error) {
	return retrieve.Result{Docs: []core.Doc{{Text: "mock"}}}, nil
}

// mockReranker is a simple mock implementation of the rerank.Reranker interface.
type mockReranker struct{}

func (m *mockReranker) Rerank(_ context.Context, req rerank.Request) ([]rerank.ScoredDoc, error) {
	var scoredDocs []rerank.ScoredDoc
	for _, doc := range req.Docs {
		scoredDocs = append(scoredDocs, rerank.ScoredDoc{Doc: doc, Score: 1.0})
	}
	return scoredDocs, nil
}

// mockLLM is a simple mock implementation of the llm.Client interface.
type mockLLM struct{}

func (m *mockLLM) Complete(_ context.Context, _ llm.Request) (llm.Response, error) {
	return llm.Response{Text: "mock"}, nil
}

// mockRules is a simple mock implementation of the core.RuleSet interface.
type mockRules struct {
	core.RuleSet
}

func (m *mockRules) Evaluate(_ core.Stage, _ core.Query, _ *core.Answer) (core.RuleResult, error) {
	return core.RuleResult{Allowed: true}, nil
}

// mockEmbedder is a simple mock implementation of the ai.Embedder interface.
type mockEmbedder struct {
}

func (m *mockEmbedder) Embed(ctx context.Context, req *ai.EmbedRequest) (*ai.EmbedResponse, error) {
	return &ai.EmbedResponse{}, nil
}

func (m *mockEmbedder) Name() string {
	return "mock-embedder"
}

func (m *mockEmbedder) Register(r api.Registry) {
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

type mockRetrieverOptions struct{}
type mockRerankerOptions struct{}
type mockLLMOptions struct{}
type mockRulesOptions struct{}
type mockEmbedderOptions struct{}
type mockStateProviderOptions struct{}

func registerTestComponents(r *Registry) {
	r.RegisterRetriever("mock-retriever", func(ctx context.Context, opts any, deps FactoryDeps) (retrieve.Retriever, error) {
		return &mockRetriever{}, nil
	})
	r.RegisterReranker("mock-reranker", func(ctx context.Context, opts any, deps FactoryDeps) (rerank.Reranker, error) {
		return &mockReranker{}, nil
	})
	r.RegisterLLM("mock-llm", func(ctx context.Context, opts any, deps FactoryDeps) (llm.Client, error) {
		return &mockLLM{}, nil
	})
	r.RegisterRuleSet("mock-rules", func(ctx context.Context, opts any, deps FactoryDeps) (core.RuleSet, error) {
		return &mockRules{}, nil
	})
	r.RegisterEmbedder("mock-embedder", func(ctx context.Context, opts any, deps FactoryDeps) (ai.Embedder, error) {
		return &mockEmbedder{}, nil
	})
	r.RegisterVectorStore("mock-vs", func(ctx context.Context, opts any, deps FactoryDeps) (core.VectorStore, error) {
		return &mockVectorStore{}, nil
	})

	r.RegisterOptions("mock-retriever", (*mockRetrieverOptions)(nil))
	r.RegisterOptions("mock-reranker", (*mockRerankerOptions)(nil))
	r.RegisterOptions("mock-llm", (*mockLLMOptions)(nil))
	r.RegisterOptions("mock-rules", (*mockRulesOptions)(nil))
	r.RegisterOptions("mock-embedder", (*mockEmbedderOptions)(nil))
	r.RegisterOptions("mock-vs", (*core.LocalvecOptions)(nil))
	r.RegisterStateProvider("mock-sp", func(ctx context.Context, opts any, deps FactoryDeps) (core.StateProvider, error) {
		return &mockStateProvider{}, nil
	})
	r.RegisterOptions("mock-sp", (*mockStateProviderOptions)(nil))
}

func TestBuilder(t *testing.T) {
	r := NewRegistry()
	registerTestComponents(r)

	b := NewBuilder(r)
	if b == nil {
		t.Fatal("expected builder to be non-nil")
	}

	b.WithRetriever(&mockRetrieverOptions{})
	b.WithReranker(&mockRerankerOptions{})
	b.WithLLM(&mockLLMOptions{})
	b.WithRules(&mockRulesOptions{})
	b.WithEmbedder(&mockEmbedderOptions{})
	b.WithVectorStore(&core.LocalvecOptions{})

	if b.retrieverName != "mock-retriever" {
		t.Errorf("expected retriever name to be mock-retriever, got %s", b.retrieverName)
	}
	if b.rerankerName != "mock-reranker" {
		t.Errorf("expected reranker name to be mock-reranker, got %s", b.rerankerName)
	}
	if b.llmName != "mock-llm" {
		t.Errorf("expected llm name to be mock-llm, got %s", b.llmName)
	}
	if b.rulesName != "mock-rules" {
		t.Errorf("expected rules name to be mock-rules, got %s", b.rulesName)
	}
	if b.embedderName != "mock-embedder" {
		t.Errorf("expected embedder name to be mock-embedder, got %s", b.embedderName)
	}
	if b.vectorStoreName != "mock-vs" {
		t.Errorf("expected vector store name to be mock-vs, got %s", b.vectorStoreName)
	}
}

func TestBuild(t *testing.T) {
	r := NewRegistry()
	registerTestComponents(r)

	b := NewBuilder(r)
	b.WithRetriever(&mockRetrieverOptions{})
	b.WithLLM(&mockLLMOptions{})
	b.WithEmbedder(&mockEmbedderOptions{})
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
	r := NewRegistry()
	registerTestComponents(r)

	builder := NewBuilder(r).
		WithRetriever(&mockRetrieverOptions{}).
		WithReranker(&mockRerankerOptions{}).
		WithLLM(&mockLLMOptions{})

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
	r := NewRegistry()
	registerTestComponents(r)

	b := NewBuilder(r)
	b.WithStateProvider(&mockStateProviderOptions{})
	if b.stateProviderName != "mock-sp" {
		t.Errorf("expected state provider name to be mock-sp, got %s", b.stateProviderName)
	}
	b.WithRetriever(&mockRetrieverOptions{})
	b.WithLLM(&mockLLMOptions{})
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
