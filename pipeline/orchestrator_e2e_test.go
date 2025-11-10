//go:build testhooks
// +build testhooks

package pipeline_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
	"github.com/duynguyendang/manglekit/internal/providers/llm"
	"github.com/duynguyendang/manglekit/internal/providers/retrievers"
	"github.com/duynguyendang/manglekit/pipeline/declarative"
	"github.com/duynguyendang/manglekit/pipeline/sandwich"
	"github.com/duynguyendang/manglekit/sdk"
	"github.com/stretchr/testify/require"
)

// mockToolHandler is a simplified handler just for the mock tool.
type mockToolHandler struct{}

func (h *mockToolHandler) Kind() core.Kind { return core.KindTool }
func (h *mockToolHandler) BuildComponent(ctx context.Context, b any, f any, resolved *core.Resolved, cfg core.ProviderOptions, name string) (core.ResourceCloser, error) {
	factory := f.(core.Factory)
	// Mock tools in this test have no dependencies.
	deps := diapi.NoopDeps{
		CoreDeps: b.(diapi.Builder).GetCoreDeps(),
	}
	comp, err := factory.Build(ctx, deps, cfg)
	if err != nil {
		return nil, err
	}
	resolved.Tools[name] = comp.(core.Tool)
	return nil, nil
}

func TestE2ESandwich_Execute(t *testing.T) {
	r := manglekit.NewRegistry()
	// Register Sandwich-specific components
	r.RegisterHandler(sandwich.NewHandler())
	r.RegisterHandler(&llm.Handler{})
	r.RegisterHandler(retrievers.NewHandler())
	sandwich.Register(r)
	manglekit.Register(r, &mockLLMOptions{}, func(ctx context.Context, deps diapi.LLMDeps, cfg *mockLLMOptions) (core.LLMClient, error) { return &mockLLM{}, nil })
	manglekit.Register(r, &mockRetrieverOptions{}, func(ctx context.Context, deps diapi.NoopDeps, cfg *mockRetrieverOptions) (core.Retriever, error) { return &mockRetriever{}, nil })

	yaml, err := os.ReadFile(filepath.Join("testdata", "config.yaml"))
	require.NoError(t, err)

	orch, err := sdk.LoadWithRegistry(context.Background(), yaml, r)
	require.NoError(t, err)
	require.NotNil(t, orch)

	// Execute the pipeline
	answer, err := orch.Execute(context.Background(), "test-session", core.Query{Text: "what is manglekit?"})
	require.NoError(t, err)
	require.Equal(t, "mock llm response", answer.Text)
}

func TestE2EDeclarative_Execute(t *testing.T) {
	r := manglekit.NewRegistry()
	// Register Declarative-specific components
	r.RegisterHandler(declarative.NewHandler())
	r.RegisterHandler(&mockToolHandler{})
	declarative.Register(r)
	manglekit.Register(r, &mockToolOptions{}, func(ctx context.Context, deps diapi.NoopDeps, cfg *mockToolOptions) (core.Tool, error) { return &mockTool{}, nil })

	yaml, err := os.ReadFile(filepath.Join("testdata", "declarative_config.yaml"))
	require.NoError(t, err)

	orch, err := sdk.LoadWithRegistry(context.Background(), yaml, r)
	require.NoError(t, err)
	require.NotNil(t, orch)

	// Execute the pipeline
	answer, err := orch.Execute(context.Background(), "test-session", core.Query{Text: "run the plan"})
	require.NoError(t, err)
	require.Contains(t, answer.Meta, "input_value_2_output")
	require.Equal(t, "mock tool output", answer.Meta["input_value_2_output"])
}

func TestE2E_ConfigValidationError(t *testing.T) {
	// Create a registry missing the LLM factory and options.
	r := manglekit.NewRegistry()
	r.RegisterHandler(sandwich.NewHandler())
	r.RegisterHandler(retrievers.NewHandler())
	sandwich.Register(r)
	manglekit.Register(r, &mockRetrieverOptions{}, func(ctx context.Context, deps diapi.NoopDeps, cfg *mockRetrieverOptions) (core.Retriever, error) { return &mockRetriever{}, nil })

	yaml, err := os.ReadFile(filepath.Join("testdata", "config.yaml"))
	require.NoError(t, err)

	_, err = sdk.LoadWithRegistry(context.Background(), yaml, r)
	require.Error(t, err)
	require.Contains(t, err.Error(), `could not find options type for kind=llm, type=mock-llm`)
}
