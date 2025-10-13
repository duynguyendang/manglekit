package manglekit

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/pipeline"
	"github.com/duynguyendang/manglekit/internal/providers/mock"
	"github.com/duynguyendang/manglekit/retrieve"
	"github.com/duynguyendang/manglekit/rerank"
	"github.com/duynguyendang/manglekit/llm"
)

func TestMain(m *testing.M) {
	RegisterRetriever("mock-retriever", func(options map[string]interface{}) (retrieve.Retriever, error) {
		pairs := make(map[string]string)
		if p, ok := options["pairs"].(map[string]interface{}); ok {
			for k, v := range p {
				pairs[k] = v.(string)
			}
		}
		return mock.NewRetriever(pairs), nil
	})
	RegisterReranker("mock-reranker", func(options map[string]interface{}) (rerank.Reranker, error) {
		passthrough := make(map[string]bool)
		if p, ok := options["passthrough"].(map[string]interface{}); ok {
			for k, v := range p {
				passthrough[k] = v.(bool)
			}
		}
		return mock.NewReranker(passthrough), nil
	})
	RegisterLLM("mock-llm", func(options map[string]interface{}) (llm.Client, error) {
		model, _ := options["model"].(string)
		return mock.NewLLM(model), nil
	})
	RegisterOptions("mock-retriever", (*mock.RetrieverOptions)(nil))
	RegisterOptions("mock-reranker", (*mock.RerankerOptions)(nil))
	RegisterOptions("mock-llm", (*mock.LLMOptions)(nil))
	os.Exit(m.Run())
}

func TestBuilderFromYAML(t *testing.T) {
	testCases := []struct {
		filename string
		want     core.Orchestrator
	}{
		{
			filename: "testdata/sandwich.yaml",
			want:     must(pipeline.NewSandwich(core.Options{
				Retriever: mock.NewRetriever(map[string]string{"foo": "bar"}),
				Reranker:          mock.NewReranker(nil),
				LLM:               mock.NewLLM("foobar"),
				FallbackThreshold: 0.5,
			})),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.filename, func(t *testing.T) {
			builder, err := NewBuilderFromYAML(tc.filename)
			if err != nil {
				t.Fatal(err)
			}
			got, err := builder.Build(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tc.want, got, cmp.AllowUnexported(pipeline.Sandwich{}, mock.Retriever{}, mock.Reranker{}, mock.LLM{})); diff != "" {
				t.Errorf("builder.Build() returned diff (-want, +got):\n%s", diff)
			}
		})
	}
}

func TestBuilderFromYAMLFail(t *testing.T) {
	testCases := []struct {
		filename    string
		expectedErr error
	}{
		{"testdata/bad-llm-provider.yaml", core.ErrInvalidOptions},
		{"testdata/bad-reranker-options.yaml", core.ErrInvalidOptions},
	}
	for _, tc := range testCases {
		t.Run(tc.filename, func(t *testing.T) {
			builder, err := NewBuilderFromYAML(tc.filename)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := builder.Build(context.Background()); err == nil {
				t.Fatalf("builder.Build() for %q succeeded, want error", tc.filename)
			}
		})
	}
}

func TestBuilderFromEnv(t *testing.T) {
	llmParams, err := json.Marshal(map[string]interface{}{"model": "foo"})
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("MKT_LLM_NAME", "mock-llm")
	t.Setenv("MKT_LLM_PARAMS", string(llmParams))

	builder, err := NewBuilderFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := builder.Build(context.Background()); err != nil {
		t.Errorf("builder.Build() failed: %v", err)
	}
}

func TestBuilderFromEnvWithYAMLFallback(t *testing.T) {
	llmParams, err := json.Marshal(map[string]interface{}{"model": "bar"})
	if err != nil {
		t.Fatal(err)
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