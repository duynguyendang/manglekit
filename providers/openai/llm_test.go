package openai

import (
	"testing"
)

func TestNewLLM_DefaultModel(t *testing.T) {
	llm, err := NewLLM("test-key", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if llm.model != "gpt-3.5-turbo" {
		t.Errorf("expected default model 'gpt-3.5-turbo', got %q", llm.model)
	}
}

func TestNewLLM_CustomModel(t *testing.T) {
	llm, err := NewLLM("test-key", "", "gpt-4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if llm.model != "gpt-4" {
		t.Errorf("expected model 'gpt-4', got %q", llm.model)
	}
}

func TestNewLLM_EmptyBaseURL(t *testing.T) {
	llm, err := NewLLM("test-key", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if llm.client == nil {
		t.Error("expected non-nil client")
	}
}

func TestNewLLM_CustomBaseURL(t *testing.T) {
	llm, err := NewLLM("test-key", "https://custom.api.com/v1", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if llm.client == nil {
		t.Error("expected non-nil client")
	}
	if llm.model != "gpt-3.5-turbo" {
		t.Errorf("expected default model, got %q", llm.model)
	}
}

func TestNewLLM_CustomModelAndBaseURL(t *testing.T) {
	llm, err := NewLLM("test-key", "https://custom.api.com/v1", "gpt-4o")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if llm.model != "gpt-4o" {
		t.Errorf("expected model 'gpt-4o', got %q", llm.model)
	}
}

func TestNewLLM_EmptyAPIKey(t *testing.T) {
	_, err := NewLLM("", "", "")
	if err != nil {
		t.Fatalf("unexpected error with empty key: %v", err)
	}
}