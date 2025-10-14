package manglekit_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/embed"
	"github.com/duynguyendang/manglekit/llm"
	"github.com/duynguyendang/manglekit/pipeline"
	"github.com/duynguyendang/manglekit/rerank"
	"github.com/duynguyendang/manglekit/retrieve"
	"github.com/firebase/genkit/go/ai"
	api "github.com/firebase/genkit/go/core/api"
	"github.com/openai/openai-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubFlowController struct {
	preResult core.RuleResult
}

func (s *stubFlowController) Evaluate(stage core.Stage, q core.Query, a *core.Answer) (core.RuleResult, error) {
	if stage == core.Pre {
		return s.preResult, nil
	}
	return core.RuleResult{Allowed: true}, nil
}

func (s *stubFlowController) Query(ctx context.Context, query string, onSolution func(map[string]any) error) error {
	return nil
}

func installStubRules(t *testing.T, result core.RuleResult) {
	manglekit.RegisterOptions("mangle", (*core.MangleOptions)(nil))
	original := manglekit.Registry.Rules["mangle"]
	manglekit.Registry.Rules["mangle"] = func(ctx context.Context, opts core.MangleOptions) (core.RuleSet, error) {
		return &stubFlowController{preResult: result}, nil
	}
	t.Cleanup(func() {
		manglekit.Registry.Rules["mangle"] = original
	})
}

type stubRetriever struct {
	docs  []core.Doc
	calls []retrieve.Request
}

func (s *stubRetriever) Retrieve(ctx context.Context, req retrieve.Request) (retrieve.Result, error) {
	s.calls = append(s.calls, req)
	copied := append([]core.Doc(nil), s.docs...)
	return retrieve.Result{Docs: copied}, nil
}

type stubEmbedder struct{}

func (stubEmbedder) Name() string { return "stub-embedder" }

func (stubEmbedder) Embed(ctx context.Context, req *ai.EmbedRequest) (*ai.EmbedResponse, error) {
	return &ai.EmbedResponse{}, nil
}

func (stubEmbedder) Register(api.Registry) {}

type stubReranker struct {
	response []rerank.ScoredDoc
	calls    []rerank.Request
}

func (s *stubReranker) Rerank(ctx context.Context, req rerank.Request) ([]rerank.ScoredDoc, error) {
	s.calls = append(s.calls, req)
	return append([]rerank.ScoredDoc(nil), s.response...), nil
}

type stubOpenAILLM struct {
	model string
	calls []llm.Request
}

func (s *stubOpenAILLM) Complete(ctx context.Context, req llm.Request) (llm.Response, error) {
	s.calls = append(s.calls, req)
	return llm.Response{Text: fmt.Sprintf("%s:%s", s.model, req.Prompt)}, nil
}

var stubFactories struct {
	retriever     *stubRetriever
	reranker      *stubReranker
	llm           *stubOpenAILLM
	retrieverOpts retrieve.InMemoryOptions
	rerankerOpts  rerank.CosineOptions
	llmOpts       llm.OpenAIOptions
}

func installStubProviders(t *testing.T) {
        manglekit.RegisterOptions("in-memory", (*retrieve.InMemoryOptions)(nil))
        manglekit.RegisterOptions("cosine", (*rerank.CosineOptions)(nil))
        manglekit.RegisterOptions("openai", (*llm.OpenAIOptions)(nil))
        manglekit.RegisterOptions("groq", (*llm.OpenAIOptions)(nil))
        manglekit.RegisterOptions("openai-embedder", (*embed.OpenAIEmbedderOptions)(nil))

        t.Setenv("GROQ_API_KEY", "test-key")
        origRetriever := manglekit.Registry.Retriever["in-memory"]
        origReranker := manglekit.Registry.Reranker["cosine"]
	origEmbedder := manglekit.Registry.Embedder["openai"]
	origLLMOpenAI := manglekit.Registry.LLM["openai"]
	origLLMGroq := manglekit.Registry.LLM["groq"]

	stubFactories = struct {
		retriever     *stubRetriever
		reranker      *stubReranker
		llm           *stubOpenAILLM
		retrieverOpts retrieve.InMemoryOptions
		rerankerOpts  rerank.CosineOptions
		llmOpts       llm.OpenAIOptions
	}{}

	manglekit.Registry.Retriever["in-memory"] = func(opts retrieve.InMemoryOptions) (retrieve.Retriever, error) {
		stubFactories.retrieverOpts = opts
		stubFactories.retriever = &stubRetriever{docs: append([]core.Doc(nil), opts.Documents...)}
		return stubFactories.retriever, nil
	}
	manglekit.Registry.Reranker["cosine"] = func(opts rerank.CosineOptions, _ ai.Embedder) (rerank.Reranker, error) {
		stubFactories.rerankerOpts = opts
		stubFactories.reranker = &stubReranker{}
		return stubFactories.reranker, nil
	}
	manglekit.Registry.Embedder["openai"] = func(opts embed.OpenAIEmbedderOptions, _ *openai.Client) (ai.Embedder, error) {
		return stubEmbedder{}, nil
	}
	llmStub := func(opts llm.OpenAIOptions, _ *openai.Client) (llm.Client, error) {
		stubFactories.llmOpts = opts
		stubFactories.llm = &stubOpenAILLM{model: opts.Model}
		return stubFactories.llm, nil
	}
	manglekit.Registry.LLM["openai"] = llmStub
	manglekit.Registry.LLM["groq"] = llmStub
	manglekit.RegisterOptions("groq", (*llm.OpenAIOptions)(nil))

	t.Cleanup(func() {
		manglekit.Registry.Retriever["in-memory"] = origRetriever
		manglekit.Registry.Reranker["cosine"] = origReranker
		manglekit.Registry.Embedder["openai"] = origEmbedder
		manglekit.Registry.LLM["openai"] = origLLMOpenAI
		manglekit.Registry.LLM["groq"] = origLLMGroq
	})
}

func requireSandwich(t *testing.T, orch core.Orchestrator) *pipeline.Sandwich {
	t.Helper()
	sandwich, ok := orch.(*pipeline.Sandwich)
	if !ok {
		t.Fatalf("expected *pipeline.Sandwich, got %T", orch)
	}
	return sandwich
}

func TestNewBuilderFromYAML_SandwichHappyPath(t *testing.T) {
	installStubProviders(t)
	installStubRules(t, core.RuleResult{Allowed: true})
	t.Setenv("OPENAI_API_KEY", "test-key")

	builder, err := manglekit.NewBuilderFromYAML("testdata/config_valid.yaml")
	if err != nil {
		t.Fatalf("NewBuilderFromYAML returned error: %v", err)
	}

	orch, err := builder.Build(context.Background())
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	t.Cleanup(func() { _ = orch.Close(context.Background()) })

	_ = requireSandwich(t, orch)
	if stubFactories.retriever == nil {
		t.Fatalf("stub retriever was not constructed")
	}
	if stubFactories.retrieverOpts.Logger == nil {
		t.Fatalf("expected retriever logger to be injected")
	}
	if child := stubFactories.retrieverOpts.Logger.With("component", "test"); child == nil {
		t.Fatalf("expected logger.With to return a logger instance")
	}
	if stubFactories.reranker == nil {
		t.Fatalf("stub reranker was not constructed")
	}
	stubFactories.reranker.response = []rerank.ScoredDoc{{Doc: stubFactories.retriever.docs[0], Score: 0.9}}

	answer, err := orch.Run(context.Background(), core.Query{Text: "hello"})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if answer.Text != "gpt-4o-mini:hello" {
		t.Fatalf("unexpected LLM answer: %q", answer.Text)
	}
	if len(stubFactories.retriever.calls) != 1 {
		t.Fatalf("expected retriever to be invoked once, got %d", len(stubFactories.retriever.calls))
	}
	if stubFactories.retriever.calls[0].TopK != 3 {
		t.Fatalf("expected TopK 3, got %d", stubFactories.retriever.calls[0].TopK)
	}
	if stubFactories.rerankerOpts.TopK != 2 {
		t.Fatalf("expected reranker topK 2, got %d", stubFactories.rerankerOpts.TopK)
	}
	if stubFactories.llm == nil {
		t.Fatalf("stub llm was not constructed")
	}
	if stubFactories.llm.model != "gpt-4o-mini" {
		t.Fatalf("unexpected llm model: %s", stubFactories.llm.model)
	}
	if len(stubFactories.llm.calls) != 1 || stubFactories.llm.calls[0].MaxTokens != 256 {
		t.Fatalf("expected MaxTokens 256 in LLM request, got calls=%v", stubFactories.llm.calls)
	}
}

func TestNewBuilderFromYAML_InvalidProvider(t *testing.T) {
	installStubRules(t, core.RuleResult{Allowed: true})
	installStubProviders(t)

	builder, err := manglekit.NewBuilderFromYAML("testdata/config_invalid_provider.yaml")
	if err != nil {
		t.Fatalf("unexpected error loading config: %v", err)
	}

	_, err = builder.Build(context.Background())
	if err == nil {
		t.Fatalf("expected error for unknown provider")
	}
	if !strings.Contains(err.Error(), "unsupported provider") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewBuilderFromYAML_InvalidOptions(t *testing.T) {
	_, err := manglekit.NewBuilderFromYAML("testdata/config_invalid_options.yaml")
	if err == nil {
		t.Fatalf("expected error for invalid options")
	}
	if !strings.Contains(err.Error(), "failed to unmarshal") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewBuilderFromYAML_EnvOverrides(t *testing.T) {
	installStubProviders(t)
	installStubRules(t, core.RuleResult{Allowed: true})
	t.Setenv("OPENAI_API_KEY", "openai-key")

	t.Setenv("EMBEDDER_MODEL", "text-embedding-3-small")
	t.Setenv("LLM_PROVIDER", "openai")
	t.Setenv("LLM_MODEL", "gpt-4o-mini")
	t.Setenv("FALLBACK_THRESHOLD", "0.1")
	t.Setenv("MAX_TOKENS", "128")
	t.Setenv("TOPK", "4")

	builder, err := manglekit.NewBuilderFromYAML("testdata/config_env_base.yaml")
	if err != nil {
		t.Fatalf("loading config failed: %v", err)
	}

	orch, err := builder.Build(context.Background())
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	orch.Close(context.Background())

	t.Setenv("LLM_PROVIDER", "groq")
	t.Setenv("LLM_MODEL", "groq-mini")

	builder, err = manglekit.NewBuilderFromYAML("testdata/config_env_base.yaml")
	if err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	orch, err = builder.Build(context.Background())
	if err != nil {
		t.Fatalf("build with overrides failed: %v", err)
	}
	defer orch.Close(context.Background())

	_ = requireSandwich(t, orch)
	if stubFactories.retriever == nil || len(stubFactories.retriever.docs) == 0 {
		t.Fatalf("expected stub retriever documents to be configured")
	}
	stubFactories.retriever.calls = nil
	stubFactories.llm.calls = nil
	if stubFactories.reranker == nil {
		t.Fatalf("stub reranker was not constructed")
	}
	stubFactories.reranker.response = []rerank.ScoredDoc{{Doc: stubFactories.retriever.docs[0], Score: 0.05}}

	_, err = orch.Run(context.Background(), core.Query{Text: "question"})
	if !errors.Is(err, core.ErrNoEvidence) {
		t.Fatalf("expected ErrNoEvidence from fallback, got %v", err)
	}
	if stubFactories.llm == nil {
		t.Fatalf("stub llm was not constructed")
	}
	if len(stubFactories.llm.calls) != 0 {
		t.Fatalf("llm should not be called when fallback triggers")
	}
	if len(stubFactories.retriever.calls) == 0 || stubFactories.retriever.calls[0].TopK != 4 {
		t.Fatalf("expected TopK override 4, got calls=%v", stubFactories.retriever.calls)
	}
	if stubFactories.llm == nil {
		t.Fatalf("stub llm was not constructed")
	}
	if stubFactories.llm.model != "groq-mini" {
		t.Fatalf("expected overridden model groq-mini, got %s", stubFactories.llm.model)
	}
}

func TestNewBuilderFromEnv(t *testing.T) {
	installStubProviders(t)
	installStubRules(t, core.RuleResult{Allowed: true})
	t.Setenv("OPENAI_API_KEY", "test-key")

	retrieverParams := `{"documents": [{"id":"env-doc","text":"Hello","source":"env"}]}`
	llmParams := `{"model":"gpt-4o-mini"}`

	t.Setenv("MKT_RETRIEVER_NAME", "in-memory")
	t.Setenv("MKT_RETRIEVER_PARAMS", retrieverParams)
	t.Setenv("MKT_LLM_NAME", "openai")
	t.Setenv("MKT_LLM_PARAMS", llmParams)
	t.Setenv("MKT_EMBEDDER_NAME", "openai")
	t.Setenv("MKT_EMBEDDER_PARAMS", `{"model":"text-embedding-3-small"}`)
	t.Setenv("MKT_RERANKER_NAME", "cosine")
	t.Setenv("MKT_RERANKER_PARAMS", `{"topK":1}`)
	t.Setenv("MKT_TOPK", "6")
	t.Setenv("MKT_MAX_TOKENS", "400")
	t.Setenv("MKT_FALLBACK_THRESHOLD", "0.7")

	builder, err := manglekit.NewBuilderFromEnv()
	if err != nil {
		t.Fatalf("NewBuilderFromEnv failed: %v", err)
	}

	orch, err := builder.Build(context.Background())
	if err != nil {
		t.Fatalf("build from env failed: %v", err)
	}
	defer orch.Close(context.Background())

	_ = requireSandwich(t, orch)
	if stubFactories.reranker == nil {
		t.Fatalf("stub reranker was not constructed")
	}
	stubFactories.reranker.response = []rerank.ScoredDoc{{Doc: core.Doc{ID: "env-doc"}, Score: 0.9}}

	if _, err := orch.Run(context.Background(), core.Query{Text: "ask"}); err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if len(stubFactories.retriever.calls) == 0 || stubFactories.retriever.calls[0].TopK != 6 {
		t.Fatalf("expected TopK 6, got calls=%v", stubFactories.retriever.calls)
	}
	if stubFactories.llm == nil {
		t.Fatalf("stub llm was not constructed")
	}
	if stubFactories.llm.model != "gpt-4o-mini" {
		t.Fatalf("unexpected llm model: %s", stubFactories.llm.model)
	}
	if len(stubFactories.llm.calls) == 0 {
		t.Fatalf("expected LLM call to be recorded")
	}
	if stubFactories.llm.calls[0].MaxTokens != 400 {
		t.Fatalf("expected MaxTokens 400 in LLM request, got calls=%v", stubFactories.llm.calls)
	}
	if stubFactories.retrieverOpts.Logger == nil {
		t.Fatalf("expected retriever logger to be injected via env config")
	}
}

func TestBuilder_ProgrammaticAPI(t *testing.T) {
	installStubProviders(t)
	t.Setenv("OPENAI_API_KEY", "test-key")

	retrieverOpts := &retrieve.InMemoryOptions{Documents: []core.Doc{{ID: "doc1", Text: "text1"}}}
	llmOpts := &llm.OpenAIOptions{Model: "gpt-4o-programmatic"}
	rerankerOpts := &rerank.CosineOptions{TopK: 5}

	builder := manglekit.NewBuilder().
		WithRetriever(retrieverOpts).
		WithLLM(llmOpts).
		WithReranker(rerankerOpts).
		WithEmbedder(&embed.OpenAIEmbedderOptions{})

	orch, err := builder.Build(context.Background())
	require.NoError(t, err)
	require.NotNil(t, orch)
	defer orch.Close(context.Background())

	_, ok := orch.(*pipeline.Sandwich)
	require.True(t, ok)

	assert.Equal(t, retrieverOpts.Documents[0].ID, stubFactories.retrieverOpts.Documents[0].ID)
	assert.Equal(t, llmOpts.Model, stubFactories.llmOpts.Model)
	assert.Equal(t, rerankerOpts.TopK, stubFactories.rerankerOpts.TopK)
}
