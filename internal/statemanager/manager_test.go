package statemanager

import (
	"context"
	"testing"

	"github.com/duynguyendang/manglekit-wip/core"
	"github.com/google/uuid"
)

// mockStateProvider is a simple in-memory state provider for testing
type mockStateProvider struct {
	data map[string]any
}

func newMockStateProvider() *mockStateProvider {
	return &mockStateProvider{
		data: make(map[string]any),
	}
}

func (m *mockStateProvider) Get(ctx context.Context, sessionID string) (any, error) {
	return m.data[sessionID], nil
}

func (m *mockStateProvider) Set(ctx context.Context, sessionID string, state any) error {
	m.data[sessionID] = state
	return nil
}

func (m *mockStateProvider) Delete(ctx context.Context, sessionID string) error {
	delete(m.data, sessionID)
	return nil
}

func (m *mockStateProvider) Close(ctx context.Context) error {
	return nil
}

// mockEvaluator is a simple mock for testing
type mockEvaluator struct {
	loadedFacts []string
}

func (m *mockEvaluator) LoadFacts(facts []string) error {
	m.loadedFacts = append(m.loadedFacts, facts...)
	return nil
}

func (m *mockEvaluator) LoadPolicy(ctx context.Context, policy string) error {
	return nil
}

func (m *mockEvaluator) AssessPlan(ctx context.Context, input core.Envelope) (core.Decision, error) {
	return core.Decision{Outcome: core.DecisionProceed}, nil
}

func (m *mockEvaluator) Assess(ctx context.Context, actionMeta core.ActionMetadata, input core.Envelope) error {
	return nil
}

func (m *mockEvaluator) Reflect(ctx context.Context, actionMeta core.ActionMetadata, output core.Envelope) (core.Envelope, error) {
	return output, nil
}

func (m *mockEvaluator) EvaluateSteering(ctx context.Context, input core.Envelope) (string, map[string]string, error) {
	return "", nil, nil
}

func (m *mockEvaluator) GetActionConfig(ctx context.Context, input core.Envelope) (map[string]string, error) {
	return nil, nil
}

func (m *mockEvaluator) CheckRequirement(ctx context.Context, env core.Envelope, requirement string) (bool, error) {
	return false, nil
}

func (m *mockEvaluator) RegisterAction(metadata core.ActionMetadata) error {
	return nil
}

func (m *mockEvaluator) Query(ctx context.Context, facts []string, queryStr string) ([]map[string]string, error) {
	return nil, nil
}

func (m *mockEvaluator) Logger() core.Logger {
	return core.NopLogger{}
}

func TestDurableStateManager_Checkpoint(t *testing.T) {
	provider := newMockStateProvider()
	engine := &mockEvaluator{}
	manager := New(provider, engine, core.NopLogger{})

	state := &core.SessionState{
		SessionID: "test-session",
		ActiveEnvelope: core.Envelope{
			ID:       uuid.New(),
			Payload:  "test payload",
			Metadata: map[string]any{"key": "value"},
		},
		ExecutionCtx: core.ExecutionContext{
			RetryCount:      1,
			FeedbackHistory: []string{"feedback1"},
		},
		LogicalFacts: []string{"fact(a, b)."},
	}

	ctx := context.Background()
	err := manager.Checkpoint(ctx, state)
	if err != nil {
		t.Fatalf("Checkpoint failed: %v", err)
	}

	// Verify data was stored
	raw, err := provider.Get(ctx, "test-session")
	if err != nil {
		t.Fatalf("Failed to get stored data: %v", err)
	}
	if raw == nil {
		t.Fatal("No data was stored")
	}
}

func TestDurableStateManager_Hydrate(t *testing.T) {
	provider := newMockStateProvider()
	engine := &mockEvaluator{}
	manager := New(provider, engine, core.NopLogger{})

	// Create and checkpoint a state
	originalState := &core.SessionState{
		SessionID: "hydrate-test",
		ActiveEnvelope: core.Envelope{
			ID:       uuid.New(),
			Payload:  "original payload",
			Metadata: map[string]any{"test": "data"},
		},
		ExecutionCtx: core.ExecutionContext{
			RetryCount:      2,
			FeedbackHistory: []string{"fb1", "fb2"},
		},
		LogicalFacts: []string{"fact1.", "fact2."},
	}

	ctx := context.Background()
	if err := manager.Checkpoint(ctx, originalState); err != nil {
		t.Fatalf("Checkpoint failed: %v", err)
	}

	// Hydrate the state
	restoredState, err := manager.Hydrate(ctx, "hydrate-test")
	if err != nil {
		t.Fatalf("Hydrate failed: %v", err)
	}
	if restoredState == nil {
		t.Fatal("Hydrate returned nil state")
	}

	// Verify state was restored correctly
	if restoredState.SessionID != originalState.SessionID {
		t.Errorf("SessionID mismatch: got %v, want %v", restoredState.SessionID, originalState.SessionID)
	}
	if restoredState.ExecutionCtx.RetryCount != originalState.ExecutionCtx.RetryCount {
		t.Errorf("RetryCount mismatch: got %v, want %v", restoredState.ExecutionCtx.RetryCount, originalState.ExecutionCtx.RetryCount)
	}
	if len(restoredState.LogicalFacts) != len(originalState.LogicalFacts) {
		t.Errorf("LogicalFacts length mismatch: got %v, want %v", len(restoredState.LogicalFacts), len(originalState.LogicalFacts))
	}

	// Verify engine was primed with facts
	if len(engine.loadedFacts) != len(originalState.LogicalFacts) {
		t.Errorf("Engine not primed correctly: got %v facts, want %v", len(engine.loadedFacts), len(originalState.LogicalFacts))
	}
}

func TestDurableStateManager_HydrateNonExistent(t *testing.T) {
	provider := newMockStateProvider()
	engine := &mockEvaluator{}
	manager := New(provider, engine, core.NopLogger{})

	ctx := context.Background()
	state, err := manager.Hydrate(ctx, "non-existent-session")
	if err != nil {
		t.Fatalf("Hydrate failed: %v", err)
	}
	if state != nil {
		t.Error("Expected nil state for non-existent session")
	}
}

func TestDurableStateManager_ExtractFacts(t *testing.T) {
	provider := newMockStateProvider()
	manager := New(provider, nil, core.NopLogger{})

	env := core.Envelope{
		ID:    uuid.New(),
		Facts: []string{"fact1.", "fact2.", "fact3."},
	}

	ctx := context.Background()
	facts, err := manager.ExtractFacts(ctx, env)
	if err != nil {
		t.Fatalf("ExtractFacts failed: %v", err)
	}

	if len(facts) != 3 {
		t.Errorf("Expected 3 facts, got %v", len(facts))
	}
}

func TestDurableStateManager_Delete(t *testing.T) {
	provider := newMockStateProvider()
	manager := New(provider, nil, core.NopLogger{})

	// Store a state
	state := &core.SessionState{
		SessionID: "delete-test",
		ActiveEnvelope: core.Envelope{
			ID: uuid.New(),
		},
	}

	ctx := context.Background()
	if err := manager.Checkpoint(ctx, state); err != nil {
		t.Fatalf("Checkpoint failed: %v", err)
	}

	// Delete it
	if err := manager.Delete(ctx, "delete-test"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify it's gone
	restored, err := manager.Hydrate(ctx, "delete-test")
	if err != nil {
		t.Fatalf("Hydrate failed: %v", err)
	}
	if restored != nil {
		t.Error("State was not deleted")
	}
}

func TestDurableStateManager_RoundTrip(t *testing.T) {
	provider := newMockStateProvider()
	engine := &mockEvaluator{}
	manager := New(provider, engine, core.NopLogger{})

	// Create a complex state
	original := &core.SessionState{
		SessionID: "roundtrip-test",
		ActiveEnvelope: core.Envelope{
			ID:             uuid.New(),
			Payload:        map[string]any{"nested": map[string]string{"key": "value"}},
			Metadata:       map[string]any{"meta1": "value1", "meta2": 42},
			SecurityLabels: []string{"label1", "label2"},
			Facts:          []string{"fact1.", "fact2."},
			ContentType:    core.TypeJSON,
		},
		ExecutionCtx: core.ExecutionContext{
			RetryCount:      3,
			FeedbackHistory: []string{"fb1", "fb2", "fb3"},
			CurrentHistory: []core.Message{
				{Role: "user", Content: "hello"},
				{Role: "assistant", Content: "hi"},
			},
		},
		LogicalFacts: []string{"derived_fact1.", "derived_fact2."},
	}

	ctx := context.Background()

	// Checkpoint
	if err := manager.Checkpoint(ctx, original); err != nil {
		t.Fatalf("Checkpoint failed: %v", err)
	}

	// Hydrate
	restored, err := manager.Hydrate(ctx, "roundtrip-test")
	if err != nil {
		t.Fatalf("Hydrate failed: %v", err)
	}

	// Verify all fields
	if restored.SessionID != original.SessionID {
		t.Error("SessionID mismatch")
	}
	if restored.ExecutionCtx.RetryCount != original.ExecutionCtx.RetryCount {
		t.Error("RetryCount mismatch")
	}
	if len(restored.ExecutionCtx.FeedbackHistory) != len(original.ExecutionCtx.FeedbackHistory) {
		t.Error("FeedbackHistory length mismatch")
	}
	if len(restored.ExecutionCtx.CurrentHistory) != len(original.ExecutionCtx.CurrentHistory) {
		t.Error("CurrentHistory length mismatch")
	}
	if len(restored.LogicalFacts) != len(original.LogicalFacts) {
		t.Error("LogicalFacts length mismatch")
	}
	if len(restored.ActiveEnvelope.SecurityLabels) != len(original.ActiveEnvelope.SecurityLabels) {
		t.Error("SecurityLabels length mismatch")
	}
}
