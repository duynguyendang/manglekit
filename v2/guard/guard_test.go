package guard

import (
	"context"
	"testing"

	"github.com/duynguyendang/manglekit/v2/core"
	"github.com/duynguyendang/manglekit/v2/engine"
)

// MockAction is a simple action for testing that echoes the input.
type MockAction struct{}

func (m *MockAction) Execute(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	return input, nil
}

func (m *MockAction) Metadata() core.ActionMetadata {
	return core.ActionMetadata{
		Name: "mock-action",
		Type: "test",
	}
}

func TestGuardedAction_Execute(t *testing.T) {
	// 1. Setup
	eng := engine.New()
	mock := &MockAction{}
	guardedAction := New(mock, eng)

	// 2. Create input
	inputPayload := "hello world"
	input := core.NewEnvelope(inputPayload)

	// 3. Execute
	output, err := guardedAction.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute() returned an unexpected error: %v", err)
	}

	// 4. Verify
	if output.ID != input.ID {
		t.Errorf("Expected output ID to be %q, got %q", input.ID, output.ID)
	}

	outputPayload, ok := output.Payload.(string)
	if !ok {
		t.Fatalf("Expected output payload to be a string, but it was not")
	}

	if outputPayload != inputPayload {
		t.Errorf("Expected output payload to be %q, got %q", inputPayload, outputPayload)
	}
}
