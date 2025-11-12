//go:build testhooks
// +build testhooks

package manglekit_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
	"github.com/firebase/genkit/go/genkit"
	"github.com/stretchr/testify/require"
)

// --- Mock Components for Testing ---

// Mock Logger
type mockLogger struct{}

func (l *mockLogger) Infof(format string, args ...any)  {}
func (l *mockLogger) Debugf(format string, args ...any) {}
func (l *mockLogger) Warnf(format string, args ...any)  {}
func (l *mockLogger) Errorf(format string, args ...any) {}
func (l *mockLogger) With(args ...any) core.Logger      { return l }

// Mock LLM
type mockLLMOptions struct{}

func (o *mockLLMOptions) ProviderName() string    { return "mock-llm" }
func (o *mockLLMOptions) ProviderKind() core.Kind { return core.KindLLM }

type mockLLMClient struct{}

func (m *mockLLMClient) Complete(ctx context.Context, req core.LLMRequest) (core.LLMResponse, error) {
	return core.LLMResponse{Text: "mocked"}, nil
}
func newMockLLM(ctx context.Context, deps diapi.LLMDeps, cfg *mockLLMOptions) (core.LLMClient, error) {
	return &mockLLMClient{}, nil
}

// Mock Retriever
type mockRetrieverOptions struct{}

func (o *mockRetrieverOptions) ProviderName() string    { return "mock-retriever" }
func (o *mockRetrieverOptions) ProviderKind() core.Kind { return core.KindRetriever }

type mockRetriever struct{}

func (m *mockRetriever) Retrieve(ctx context.Context, req core.RetrieveRequest) (core.RetrieveResult, error) {
	return core.RetrieveResult{}, nil
}
func newMockRetriever(ctx context.Context, deps diapi.NoopDeps, cfg *mockRetrieverOptions) (core.Retriever, error) {
	return &mockRetriever{}, nil
}

// Mock Sandwich Orchestrator
type mockSandwichOptions struct {
	Retriever string `yaml:"retriever"`
	LLM       string `yaml:"llm"`
}

func (o *mockSandwichOptions) ProviderName() string    { return "sandwich" }
func (o *mockSandwichOptions) ProviderKind() core.Kind { return core.KindOrchestrator }

type mockSandwichDeps struct {
	diapi.CoreDeps
	Retriever core.Retriever
	LLM       core.LLMClient
}

type mockOrchestrator struct{}

func (m *mockOrchestrator) Execute(ctx context.Context, sessionID string, q core.Query) (core.Answer, error) {
	return core.Answer{}, nil
}
func (m *mockOrchestrator) Close(ctx context.Context) error { return nil }

func newMockOrchestrator(ctx context.Context, deps *mockSandwichDeps, cfg *mockSandwichOptions) (core.Orchestrator, error) {
	return &mockOrchestrator{}, nil
}

// mockComponentHandler is a generic handler for testing.
type mockComponentHandler struct {
	kind core.Kind
}

func (h *mockComponentHandler) Kind() core.Kind { return h.kind }
func (h *mockComponentHandler) BuildComponent(ctx context.Context, b any, f any, resolved *core.Resolved, cfg core.ProviderOptions, name string) (core.ResourceCloser, error) {
	builder := b.(diapi.Builder)
	var deps any
	var err error

	// Manual dependency resolution for the test
	switch c := cfg.(type) {
	case *mockSandwichOptions:
		var r core.Retriever
		var l core.LLMClient
		r, err = builder.GetRetriever(c.Retriever)
		if err != nil {
			return nil, fmt.Errorf("failed to get retriever dependency for mock sandwich: %w", err)
		}
		l, err = builder.GetLLMClient(c.LLM)
		if err != nil {
			return nil, fmt.Errorf("failed to get llm dependency for mock sandwich: %w", err)
		}
		deps = &mockSandwichDeps{CoreDeps: builder.GetCoreDeps(), Retriever: r, LLM: l}
	case *mockLLMOptions:
		deps = diapi.LLMDeps{CoreDeps: builder.GetCoreDeps(), Genkit: builder.Genkit()}
	case *mockRetrieverOptions:
		deps = diapi.NoopDeps{CoreDeps: builder.GetCoreDeps()}
	default:
		return nil, fmt.Errorf("unsupported options type in mock handler: %T", cfg)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to resolve deps for %s: %w", name, err)
	}

	factory, ok := f.(core.Factory)
	if !ok {
		return nil, fmt.Errorf("invalid factory type: %T", f)
	}
	comp, err := factory.Build(ctx, deps, cfg)
	if err != nil {
		return nil, fmt.Errorf("mock component factory failed: %w", err)
	}

	switch h.kind {
	case core.KindOrchestrator:
		resolved.Orchestrators[name] = comp.(core.Orchestrator)
	case core.KindLLM:
		resolved.LLMs[name] = comp.(core.LLMClient)
	case core.KindRetriever:
		resolved.Retrievers[name] = comp.(core.Retriever)
	}

	return nil, nil
}

// testRegistryWithMocks creates a new registry and populates it with all the mock components and their handlers.
func testRegistryWithMocks() *manglekit.Registry {
	r := manglekit.NewRegistry()
	// 3-Part Registration Rule: Options, Factory, and Handler
	manglekit.Register(r, &mockLLMOptions{}, newMockLLM)
	r.RegisterHandler(&mockComponentHandler{kind: core.KindLLM})

	manglekit.Register(r, &mockRetrieverOptions{}, newMockRetriever)
	r.RegisterHandler(&mockComponentHandler{kind: core.KindRetriever})

	manglekit.Register(r, &mockSandwichOptions{}, newMockOrchestrator)
	r.RegisterHandler(&mockComponentHandler{kind: core.KindOrchestrator})
	return r
}

func TestSuccessfulBuild(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	g := genkit.Init(ctx)

	registry := testRegistryWithMocks()
	obs := core.Observability{Logger: &mockLogger{}}
	builder, err := manglekit.NewBuilder(ctx, registry, obs, g)
	require.NoError(t, err)

	builder.WithOptions("my-retriever", &mockRetrieverOptions{})
	builder.WithOptions("my-llm", &mockLLMOptions{})
	builder.WithOptions("my-sandwich", &mockSandwichOptions{
		Retriever: "my-retriever",
		LLM:       "my-llm",
	})

	orch, _, err := builder.Build(ctx, "my-sandwich", "")
	require.NoError(t, err)
	require.NotNil(t, orch)
}

func TestMissingDependencyError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	g := genkit.Init(ctx)

	// Register only the orchestrator and its handler.
	registry := manglekit.NewRegistry()
	manglekit.Register(registry, &mockSandwichOptions{}, newMockOrchestrator)
	registry.RegisterHandler(&mockComponentHandler{kind: core.KindOrchestrator})

	obs := core.Observability{Logger: &mockLogger{}}
	builder, err := manglekit.NewBuilder(ctx, registry, obs, g)
	require.NoError(t, err)

	builder.WithOptions("my-sandwich-broken", &mockSandwichOptions{
		Retriever: "i-do-not-exist",
		LLM:       "i-do-not-exist-either",
	})

	_, _, err = builder.Build(ctx, "my-sandwich-broken", "")

	require.Error(t, err)
	require.Contains(t, err.Error(), "dependency not found: i-do-not-exist")
}
