package ai

import (
	"context"
	"errors"
	"testing"

	"github.com/duynguyendang/manglekit/core"
)

// mockTextGenerator is a test implementation of TextGenerator.
type mockTextGenerator struct {
	response string
	err      error
}

func (m *mockTextGenerator) Complete(ctx context.Context, prompt string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.response, nil
}

func TestLLMAction_Execute_Success(t *testing.T) {
	generator := &mockTextGenerator{response: "Generated response"}
	action, err := NewLLMAction("test-llm", generator)
	if err != nil {
		t.Fatalf("unexpected error creating action: %v", err)
	}

	input := core.NewEnvelope("Test prompt")
	output, err := action.Execute(context.Background(), input)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	payload, ok := output.Payload.(string)
	if !ok {
		t.Fatalf("expected string payload, got %T", output.Payload)
	}

	if payload != "Generated response" {
		t.Errorf("expected 'Generated response', got %q", payload)
	}

	if output.GetMeta("model_type") != "llm" {
		t.Errorf("expected model_type 'llm', got %q", output.GetMeta("model_type"))
	}

	if output.GetMeta("action_name") != "test-llm" {
		t.Errorf("expected action_name 'test-llm', got %q", output.GetMeta("action_name"))
	}
}

func TestLLMAction_Execute_InvalidInput(t *testing.T) {
	generator := &mockTextGenerator{response: "Response"}
	action, err := NewLLMAction("test-llm", generator)
	if err != nil {
		t.Fatalf("unexpected error creating action: %v", err)
	}

	// Pass non-string payload
	input := core.NewEnvelope(123)
	_, err = action.Execute(context.Background(), input)

	if err == nil {
		t.Fatal("expected error for invalid input type")
	}

	if !errors.Is(err, core.ErrSystemError) {
		t.Errorf("expected ErrSystemError, got %v", err)
	}
}

func TestLLMAction_Execute_GeneratorError(t *testing.T) {
	generatorErr := errors.New("generation failed")
	generator := &mockTextGenerator{err: generatorErr}
	action, err := NewLLMAction("test-llm", generator)
	if err != nil {
		t.Fatalf("unexpected error creating action: %v", err)
	}

	input := core.NewEnvelope("Test prompt")
	_, err = action.Execute(context.Background(), input)

	if err == nil {
		t.Fatal("expected error from generator")
	}

	if !errors.Is(err, generatorErr) {
		t.Errorf("expected wrapped generator error, got %v", err)
	}
}

func TestLLMAction_Metadata(t *testing.T) {
	generator := &mockTextGenerator{}
	action, err := NewLLMAction("my-llm-action", generator)
	if err != nil {
		t.Fatalf("unexpected error creating action: %v", err)
	}

	meta := action.Metadata()

	if meta.Name != "my-llm-action" {
		t.Errorf("expected name 'my-llm-action', got %q", meta.Name)
	}

	if meta.Type != "llm" {
		t.Errorf("expected type 'llm', got %q", meta.Type)
	}
}
