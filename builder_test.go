package manglekit

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/firebase/genkit/go/ai"
	api "github.com/firebase/genkit/go/core/api"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/embed"
	"github.com/duynguyendang/manglekit/llm"
	"github.com/duynguyendang/manglekit/pipeline"
	"github.com/duynguyendang/manglekit/rerank"
	"github.com/duynguyendang/manglekit/retrieve"
	"github.com/openai/openai-go"
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
	original := Registry.Rules["mangle"]
	Registry.Rules["mangle"] = func(ctx context.Context, opts core.MangleOptions) (core.RuleSet, error) {
		return &stubFlowController{preResult: result}, nil
	}
	t.Cleanup(func() {
		Registry.Rules["mangle"] = original
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
	origRetriever := Registry.Retriever["in-memory"]
	origReranker := Registry.Reranker["cosine"]
	origEmbedder := Registry.Embedder["openai"]
	origLLMOpenAI := Registry.LLM["openai"]
	origLLMGroq := Registry.LLM["groq"]

	stubFactories = struct {
		retriever     *stubRetriever
		reranker      *stubReranker
		llm           *stubOpenAILLM
		retrieverOpts retrieve.InMemoryOptions
		rerankerOpts  rerank.CosineOptions
		llmOpts       llm.OpenAIOptions
	}{}

	Registry.Retriever["in-memory"] = func(opts retrieve.InMemoryOptions) (retrieve.Retriever, error) {
		stubFactories.retrieverOpts = opts
		stubFactories.retriever = &stubRetriever{docs: append([]core.Doc(nil), opts.Documents...)}
		return stubFactories.retriever, nil
	}
	Registry.Reranker["cosine"] = func(opts rerank.CosineOptions, _ ai.Embedder) (rerank.Reranker, error) {
		stubFactories.rerankerOpts = opts
		stubFactories.reranker = &stubReranker{}
		return stubFactories.reranker, nil
	}
	Registry.Embedder["openai"] = func(opts embed.OpenAIEmbedderOptions, _ *openai.Client) (ai.Embedder, error) {
		return stubEmbedder{}, nil
	}
	llmStub := func(opts llm.OpenAIOptions, _ *openai.Client) (llm.Client, error) {
		stubFactories.llmOpts = opts
		stubFactories.llm = &stubOpenAILLM{model: opts.Model}
		return stubFactories.llm, nil
	}
	Registry.LLM["openai"] = llmStub
	Registry.LLM["groq"] = llmStub

	t.Cleanup(func() {
		Registry.Retriever["in-memory"] = origRetriever
		Registry.Reranker["cosine"] = origReranker
		Registry.Embedder["openai"] = origEmbedder
		Registry.LLM["openai"] = origLLMOpenAI
		Registry.LLM["groq"] = origLLMGroq
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

	builder, err := NewBuilderFromYAML("testdata/config_valid.yaml")
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

	builder, err := NewBuilderFromYAML("testdata/config_invalid_provider.yaml")
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
	_, err := NewBuilderFromYAML("testdata/config_invalid_options.yaml")
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

	builder, err := NewBuilderFromYAML("testdata/config_env_base.yaml")
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

	builder, err = NewBuilderFromYAML("testdata/config_env_base.yaml")
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

	builder, err := NewBuilderFromEnv()
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
}
	"encoding/json"
	"os"
	"testing"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/embed"
	"github.com/duynguyendang/manglekit/internal/providers/mock"
	"github.com/duynguyendang/manglekit/llm"
	"github.com/duynguyendang/manglekit/pipeline"
	"github.com/duynguyendang/manglekit/rerank"
	"github.com/duynguyendang/manglekit/retrieve"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMain sets up mock providers for use in all tests within this package.
func TestMain(m *testing.M) {
	RegisterRetriever("mock-retriever", func(options map[string]interface{}) (retrieve.Retriever, error) {
		pairs := make(map[string]string)
		if p, ok := options["pairs"].(map[string]interface{}); ok {
			for k, v := range p {
				if s, ok := v.(string); ok {
					pairs[k] = s
				}
			}
		}
		return mock.NewRetriever(pairs), nil
	})
	RegisterReranker("mock-reranker", func(options map[string]interface{}) (rerank.Reranker, error) {
		passthrough := make(map[string]bool)
		if p, ok := options["passthrough"].(map[string]interface{}); ok {
			for k, v := range p {
				if b, ok := v.(bool); ok {
					passthrough[k] = b
				}
			}
		}
		return mock.NewReranker(passthrough), nil
	})
	RegisterLLM("mock-llm", func(options map[string]interface{}) (llm.Client, error) {
		model, _ := options["model"].(string)
		return mock.NewLLM(model), nil
	})
	// A mock google embedder for testing.
	RegisterEmbedder("google", func(opts embed.GoogleEmbedderOptions, g *genkit.Genkit) (ai.Embedder, error) {
		return &mock.Embedder{}, nil
	})

	RegisterOptions("mock-retriever", (*mock.RetrieverOptions)(nil))
	RegisterOptions("mock-reranker", (*mock.RerankerOptions)(nil))
	RegisterOptions("mock-llm", (*mock.LLMOptions)(nil))
	RegisterOptions("google-embedder", (*embed.GoogleEmbedderOptions)(nil))

	os.Exit(m.Run())
}

func TestBuilderFromYAML(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		filename     string
		validateFunc func(t *testing.T, orch core.Orchestrator)
		expectErr    bool
		errAssert    func(t *testing.T, err error)
	}{
		{
			name:     "Valid sandwich config",
			filename: "testdata/sandwich.yaml",
			validateFunc: func(t *testing.T, orch core.Orchestrator) {
				s, ok := orch.(*pipeline.Sandwich)
				require.True(t, ok)

				llmClient := s.LLM()
				require.NotNil(t, llmClient)
				mockLLM, ok := llmClient.(*mock.LLM)
				require.True(t, ok)
				assert.Equal(t, "foobar", mockLLM.Model())

				retriever := s.Retriever()
				require.NotNil(t, retriever)
				mockRetriever, ok := retriever.(retrieve.Retriever)
				require.True(t, ok)
				res, err := mockRetriever.Retrieve(context.Background(), retrieve.Request{Query: "foo"})
				require.NoError(t, err)
				require.Len(t, res.Docs, 1)
				assert.Equal(t, "bar", res.Docs[0].Text)
			},
		},
		{
			name:      "Invalid LLM provider",
			filename:  "testdata/bad-llm-provider.yaml",
			expectErr: true,
			errAssert: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, core.ErrInvalidOptions)
				assert.Contains(t, err.Error(), "unknown llm provider")
			},
		},
		{
			name:      "Invalid reranker options",
			filename:  "testdata/bad-reranker-options.yaml",
			expectErr: true,
			errAssert: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, core.ErrInvalidOptions)
			},
		},
		{
			name:      "Non-existent file",
			filename:  "testdata/non-existent.yaml",
			expectErr: true, // This error happens at NewBuilderFromYAML time
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			builder, err := NewBuilderFromYAML(tc.filename)
			if tc.name == "Non-existent file" {
				require.Error(t, err)
				return
			}
			if tc.name == "Invalid reranker options" {
				require.Error(t, err)
				tc.errAssert(t, err)
				return
			}
			require.NoError(t, err)

			orch, err := builder.Build(context.Background())

			if tc.expectErr {
				require.Error(t, err)
				if tc.errAssert != nil {
					tc.errAssert(t, err)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, orch)
				if tc.validateFunc != nil {
					tc.validateFunc(t, orch)
				}
			}
		})
	}
}

func TestBuilderFromEnv(t *testing.T) {
	// Do not run in parallel, as t.Setenv is not safe for concurrent use.
	testCases := []struct {
		name      string
		envVars   map[string]string
		yamlFile  string // Optional: for testing env override
		validate  func(t *testing.T, orch core.Orchestrator)
		expectErr bool
	}{
		{
			name: "Basic build from env",
			envVars: map[string]string{
				"MANGLEKIT_LLM_PROVIDER": "mock-llm",
				"MANGLEKIT_LLM_OPTIONS":  `{"model": "env-model"}`,
			},
			validate: func(t *testing.T, orch core.Orchestrator) {
				s, ok := orch.(*pipeline.Sandwich)
				require.True(t, ok)
				llmClient := s.LLM()
				require.NotNil(t, llmClient)
				mockLLM, ok := llmClient.(*mock.LLM)
				require.True(t, ok)
				assert.Equal(t, "env-model", mockLLM.Model())
				retriever := s.Retriever()
				assert.Nil(t, retriever, "Retriever should not be configured from env")
			},
		},
		{
			name: "Env overrides YAML",
			envVars: map[string]string{
				"MANGLEKIT_LLM_PROVIDER": "mock-llm",
				"MANGLEKIT_LLM_OPTIONS":  `{"model": "env-override-model"}`,
			},
			yamlFile: "testdata/sandwich.yaml", // This file has model: "foobar"
			validate: func(t *testing.T, orch core.Orchestrator) {
				s, ok := orch.(*pipeline.Sandwich)
				require.True(t, ok)
				// LLM should be from env
				llmClient := s.LLM()
				require.NotNil(t, llmClient)
				mockLLM, ok := llmClient.(*mock.LLM)
				require.True(t, ok)
				assert.Equal(t, "env-override-model", mockLLM.Model())
				// Retriever should be from the YAML file
				retriever := s.Retriever()
				require.NotNil(t, retriever)
				_, ok = retriever.(retrieve.Retriever)
				assert.True(t, ok)
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			for key, value := range tc.envVars {
				t.Setenv(key, value)
			}

			var orch core.Orchestrator
			var err error
			if tc.yamlFile != "" {
				builder, err := NewBuilderFromYAML(tc.yamlFile)
				require.NoError(t, err)
				orch, err = builder.Build(context.Background())
			} else {
				builder, err := NewBuilderFromEnv()
				require.NoError(t, err)
				orch, err = builder.Build(context.Background())
			}

			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, orch)
				if tc.validate != nil {
					tc.validate(t, orch)
				}
			}
		})
	}
}

func TestGoogleEmbedderAlias(t *testing.T) {
	t.Setenv("GOOGLE_API_KEY", "test-key")

	builder := NewBuilder()
	builder.WithEmbedder(&embed.GoogleEmbedderOptions{Model: "embedding-001"})

	// This would have failed before the alias normalization.
	_, err := builder.Build(context.Background())
	require.NoError(t, err, "should build successfully with google-embedder alias")
}
	t.Setenv("MKT_LLM_NAME", "mock-llm")
	t.Setenv("MKT_LLM_PARAMS", string(llmParams))

	builder, err := NewBuilderFromYAML(filepath.Join("testdata/sandwich.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	m, err := builder.Build(context.Background())
	if err != nil {
		t.Errorf("builder.Build() failed: %v", err)
	}
	if m == nil {
		t.Fatal("builder.Build() returned nil")
	}
	res, err := m.Retriever().(*mock.Retriever).Retrieve(context.Background(), retrieve.Request{Query: "foo"})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := res.Docs[0].Text, "bar"; got != want {
		t.Errorf("m.Retriever().Retrieve() = %q, want %q", got, want)
	}
}

func must(o core.Orchestrator, err error) core.Orchestrator {
	if err != nil {
		panic(err)
	}
	return o
}
