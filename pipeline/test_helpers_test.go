//go:build testhooks
// +build testhooks

package pipeline_test

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/core"
)

// --- Mock Retriever ---

type mockRetriever struct{}

func (m *mockRetriever) Retrieve(ctx context.Context, req core.RetrieveRequest) (core.RetrieveResult, error) {
	return core.RetrieveResult{
		Docs: []core.Doc{
			{Text: "mock document"},
		},
	}, nil
}

type mockRetrieverOptions struct{}

func (m *mockRetrieverOptions) ProviderName() string    { return "mock-retriever" }
func (m *mockRetrieverOptions) ProviderKind() core.Kind { return core.KindRetriever }
func (m *mockRetrieverOptions) GetProviderOptions() any { return m }

// --- Mock LLM ---

type mockLLM struct{}

func (l *mockLLM) Complete(ctx context.Context, req core.LLMRequest) (core.LLMResponse, error) {
	return core.LLMResponse{Text: "mock llm response"}, nil
}

type mockLLMOptions struct{}

func (o *mockLLMOptions) ProviderName() string    { return "mock-llm" }
func (o *mockLLMOptions) ProviderKind() core.Kind { return core.KindLLM }
func (o *mockLLMOptions) GetProviderOptions() any { return o }

// --- Mock Tool ---

type mockTool struct {
	// ExecuteFunc allows overriding the Execute method for specific tests.
	ExecuteFunc func(ctx context.Context, execCtx *core.ExecutionContext) error
}

func (t *mockTool) Execute(ctx context.Context, execCtx *core.ExecutionContext) error {
	if t.ExecuteFunc != nil {
		return t.ExecuteFunc(ctx, execCtx)
	}
	// Default behavior: store the input key as the output in the Answer's Meta map.
	input, ok := execCtx.CurrentStepParams["input_key"].(string)
	if !ok {
		return fmt.Errorf("input_key not found or not a string")
	}
	if execCtx.Answer.Meta == nil {
		execCtx.Answer.Meta = make(map[string]any)
	}
	outputKey := fmt.Sprintf("%s_output", input)
	execCtx.Answer.Meta[outputKey] = "mock tool output"
	return nil
}

type mockToolOptions struct{}

func (o *mockToolOptions) ProviderName() string    { return "mock-tool" }
func (o *mockToolOptions) ProviderKind() core.Kind { return core.KindTool }
func (o *mockToolOptions) GetProviderOptions() any { return o }

// --- Mock State Provider ---

type mockStateProvider struct {
	store map[string]any
}

func (s *mockStateProvider) Get(ctx context.Context, sessionID string) (any, error) {
	return s.store[sessionID], nil
}

func (s *mockStateProvider) Set(ctx context.Context, sessionID string, state any) error {
	if s.store == nil {
		s.store = make(map[string]any)
	}
	s.store[sessionID] = state
	return nil
}

func (s *mockStateProvider) Delete(ctx context.Context, sessionID string) error {
	delete(s.store, sessionID)
	return nil
}

func (s *mockStateProvider) Close(ctx context.Context) error {
	return nil
}

type mockStateProviderOptions struct{}

func (o *mockStateProviderOptions) ProviderName() string    { return "mock-state-provider" }
func (o *mockStateProviderOptions) ProviderKind() core.Kind { return core.KindStateProvider }
func (o *mockStateProviderOptions) GetProviderOptions() any { return o }
