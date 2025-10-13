package manglekit

import (
	"context"
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
