package extractor

import (
	"context"
	"strings"
	"testing"

	"github.com/duynguyendang/manglekit-wip/core"
)

// MockLLM is a simple mock for core.Action.
type MockLLM struct {
	Response string
}

func (m *MockLLM) Execute(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	return core.NewEnvelope(m.Response), nil
}

func (m *MockLLM) Metadata() core.ActionMetadata {
	return core.ActionMetadata{Name: "mock-llm", Type: "llm"}
}

type TestStruct struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

func TestExtractorAction_Execute(t *testing.T) {
	// 1. Setup Mock LLM with expected JSON response
	expectedJSON := `{"name": "test_item", "value": 42}`
	mockLLM := &MockLLM{Response: expectedJSON}

	// 2. Init Extractor
	target := TestStruct{}
	extractor, err := New("test_extractor", mockLLM, target)
	if err != nil {
		t.Fatalf("Failed to create extractor: %v", err)
	}

	// 3. Execute
	input := core.NewEnvelope("Extract info about test_item with value 42")
	result, err := extractor.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// 4. Assert Output Type
	payload, ok := result.Payload.(TestStruct)
	if !ok {
		t.Fatalf("Expected payload type TestStruct, got %T", result.Payload)
	}

	// 5. Assert Values
	if payload.Name != "test_item" {
		t.Errorf("Expected Name 'test_item', got '%s'", payload.Name)
	}
	if payload.Value != 42 {
		t.Errorf("Expected Value 42, got %d", payload.Value)
	}
}

func TestExtractorAction_Execute_InvalidJSON(t *testing.T) {
	// 1. Setup Mock LLM with invalid JSON
	mockLLM := &MockLLM{Response: `invalid json`}

	// 2. Init Extractor
	extractor, err := New("test_extractor", mockLLM, TestStruct{})
	if err != nil {
		t.Fatalf("Failed to create extractor: %v", err)
	}

	// 3. Execute
	input := core.NewEnvelope("some input")
	_, err = extractor.Execute(context.Background(), input)

	// 4. Assert Error
	if err == nil {
		t.Fatal("Expected error due to invalid JSON, got nil")
	}
}

func TestExtractorAction_Execute_InvalidInput(t *testing.T) {
	mockLLM := &MockLLM{Response: `{}`}
	extractor, err := New("test_extractor", mockLLM, TestStruct{})
	if err != nil {
		t.Fatalf("Failed to create extractor: %v", err)
	}

	// Pass integer payload instead of string
	input := core.NewEnvelope(123)
	_, err = extractor.Execute(context.Background(), input)

	if err == nil {
		t.Fatal("Expected error due to invalid input type, got nil")
	}
}

func TestNew_SchemaGeneration(t *testing.T) {
	mockLLM := &MockLLM{}

	// Test that New generates schema string
	extractor, err := New("schema_test", mockLLM, TestStruct{})
	if err != nil {
		t.Fatalf("Failed to create extractor: %v", err)
	}

	if extractor.schemaStr == "" {
		t.Error("Expected schemaStr to be populated, got empty string")
	}

	// Check that schema contains field names
	if !strings.Contains(extractor.schemaStr, "name") || !strings.Contains(extractor.schemaStr, "value") {
		t.Errorf("Schema definition missing expected fields: %s", extractor.schemaStr)
	}
}
