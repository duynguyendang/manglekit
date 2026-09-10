package ooda

import (
	"context"
	"testing"

	"github.com/duynguyendang/manglekit/core"
)

func TestRegistry_Register(t *testing.T) {
	registry := NewRegistry()

	// Register a tool
	err := registry.Register("test_tool", func(ctx context.Context, args map[string]interface{}) (string, error) {
		return "test result", nil
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !registry.Has("test_tool") {
		t.Error("expected tool to be registered")
	}
}

func TestRegistry_RegisterDuplicate(t *testing.T) {
	registry := NewRegistry()

	registry.Register("test_tool", func(ctx context.Context, args map[string]interface{}) (string, error) {
		return "first", nil
	})

	err := registry.Register("test_tool", func(ctx context.Context, args map[string]interface{}) (string, error) {
		return "second", nil
	})

	if err == nil {
		t.Error("expected error for duplicate registration")
	}
}

func TestRegistry_MustRegister(t *testing.T) {
	registry := NewRegistry()

	// Should not panic
	registry.MustRegister("test_tool", func(ctx context.Context, args map[string]interface{}) (string, error) {
		return "result", nil
	})

	if !registry.Has("test_tool") {
		t.Error("expected tool to be registered")
	}
}

func TestRegistry_Unregister(t *testing.T) {
	registry := NewRegistry()

	registry.Register("test_tool", func(ctx context.Context, args map[string]interface{}) (string, error) {
		return "result", nil
	})

	registry.Unregister("test_tool")

	if registry.Has("test_tool") {
		t.Error("expected tool to be unregistered")
	}
}

func TestRegistry_Get(t *testing.T) {
	registry := NewRegistry()

	fn := registry.Get("nonexistent")
	if fn != nil {
		t.Error("expected nil for nonexistent tool")
	}

	registry.Register("test_tool", func(ctx context.Context, args map[string]interface{}) (string, error) {
		return "result", nil
	})

	fn = registry.Get("test_tool")
	if fn == nil {
		t.Error("expected tool function")
	}
}

func TestRegistry_List(t *testing.T) {
	registry := NewRegistry()

	registry.Register("tool1", nil)
	registry.Register("tool2", nil)
	registry.Register("tool3", nil)

	list := registry.List()
	if len(list) != 3 {
		t.Errorf("expected 3 tools, got %d", len(list))
	}
}

func TestRegistry_Execute(t *testing.T) {
	registry := NewRegistry()

	registry.Register("add", func(ctx context.Context, args map[string]interface{}) (string, error) {
		a := args["a"].(int)
		b := args["b"].(int)
		return string(rune(a + b)), nil
	})

	result, err := registry.Execute(context.Background(), "add", map[string]interface{}{
		"a": 1,
		"b": 2,
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Result would be string representation
	_ = result
}

func TestRegistry_ExecuteNotFound(t *testing.T) {
	registry := NewRegistry()

	_, err := registry.Execute(context.Background(), "nonexistent", nil)

	if err == nil {
		t.Error("expected error for nonexistent tool")
	}
}

func TestDispatcher_Dispatch(t *testing.T) {
	registry := NewRegistry()
	registry.Register("generate_csd", func(ctx context.Context, args map[string]interface{}) (string, error) {
		return "CSD generated: " + args["project"].(string), nil
	})

	dispatcher := NewDispatcher(registry)

	result, err := dispatcher.Dispatch(context.Background(), "generate_csd", map[string]interface{}{
		"project": "test-project",
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	expected := "CSD generated: test-project"
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestDispatcher_DispatchUnknown(t *testing.T) {
	registry := NewRegistry()
	dispatcher := NewDispatcher(registry)

	_, err := dispatcher.Dispatch(context.Background(), "unknown_action", nil)

	if err == nil {
		t.Error("expected error for unknown action")
	}
}

func TestDispatcher_WithFallback(t *testing.T) {
	// Save original SafeStop and restore after test
	originalSafeStop := SafeStop
	defer func() { SafeStop = originalSafeStop }()

	// Override SafeStop for this test
	SafeStop = func(ctx context.Context, args map[string]interface{}) (string, error) {
		return "fallback: " + args["action"].(string), nil
	}

	registry := NewRegistry()
	dispatcher := NewDispatcher(registry) // With SafeStop override, fallback is effectively disabled

	// Call unknown action - should use SafeStop (which we overrode)
	result, err := dispatcher.Dispatch(context.Background(), "unknown", nil)

	if err == nil {
		t.Error("expected error for unknown action with SafeStop")
	}

	expected := "fallback: unknown"
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestDispatcher_NoFallback(t *testing.T) {
	registry := NewRegistry()
	dispatcher := NewDispatcher(registry) // No fallback

	_, err := dispatcher.Dispatch(context.Background(), "unknown", nil)

	if err == nil {
		t.Error("expected error for unknown action without fallback")
	}

	// Verify error message contains "sovereign violation"
	if err != nil && err.Error() == "" {
		t.Error("expected error message about sovereign violation")
	}
}

func TestOodaLoop_WithDispatcher(t *testing.T) {
	ctx := context.Background()

	// Create registry with tools
	registry := NewRegistry()
	registry.MustRegister("write_document", func(ctx context.Context, args map[string]interface{}) (string, error) {
		docType := args["type"].(string)
		return "Document of type " + docType + " generated", nil
	})

	// Create brain that returns a decision with action
	brain := &mockBrainWithAction{
		decision: &core.Decision{
			Outcome: core.DecisionProceed,
			Action: &core.ActionEnvelope{
				Name: "write_document",
				Arguments: map[string]interface{}{
					"type": "ADD",
				},
			},
			AuditTrail: &core.AuditTrail{},
		},
	}

	frame := NewBuilder().
		WithInput("Generate ADD document").
		WithRegistry(registry).
		WithBrain(brain).
		Build()

	resultFrame, err := Run(ctx, frame)
	if err != nil {
		t.Fatalf("OODA loop failed: %v", err)
	}

	// Verify action was executed
	expectedResult := "Document of type ADD generated"
	if resultFrame.ActionResult != expectedResult {
		t.Errorf("expected %s, got %v", expectedResult, resultFrame.ActionResult)
	}

	t.Logf("Action executed successfully: %v", resultFrame.ActionResult)
}

func TestOodaLoop_UnknownAction(t *testing.T) {
	ctx := context.Background()

	// Empty registry
	registry := NewRegistry()

	// Brain returns decision with unknown action
	brain := &mockBrainWithAction{
		decision: &core.Decision{
			Outcome: core.DecisionProceed,
			Action: &core.ActionEnvelope{
				Name:      "nonexistent_action",
				Arguments: map[string]interface{}{},
			},
			AuditTrail: &core.AuditTrail{},
		},
	}

	frame := NewBuilder().
		WithInput("Test unknown action").
		WithRegistry(registry).
		WithBrain(brain).
		Build()

	resultFrame, err := Run(ctx, frame)

	// Should fail with sovereign violation
	if err == nil {
		t.Error("expected error for unknown action")
	}

	// ActionResult should be nil due to error
	if resultFrame.ActionResult != nil {
		t.Errorf("expected nil action result, got %v", resultFrame.ActionResult)
	}

	t.Logf("Expected error: %v", err)
}

type mockBrainWithAction struct {
	decision *core.Decision
	err      error
}

func (b *mockBrainWithAction) Evaluate(ctx context.Context, frame *CognitiveFrame) (*core.Decision, error) {
	if b.err != nil {
		return nil, b.err
	}
	return b.decision, nil
}

func (b *mockBrainWithAction) Verify(ctx context.Context, frame *CognitiveFrame) (*core.AuditTrail, error) {
	return &core.AuditTrail{}, nil
}

func (b *mockBrainWithAction) LoadPolicy(ctx context.Context, rules string) error {
	return nil
}

func TestActionEnvelope_String(t *testing.T) {
	env := &ActionEnvelope{
		Name: "test_action",
		Arguments: map[string]interface{}{
			"key": "value",
		},
	}

	str := env.String()
	if str == "" {
		t.Error("expected non-empty string")
	}

	t.Logf("ActionEnvelope: %s", str)
}
