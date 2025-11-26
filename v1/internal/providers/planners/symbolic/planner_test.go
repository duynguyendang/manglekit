package symbolic

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/duynguyendang/manglekit/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockLogger is a simple logger for testing.
type mockLogger struct{}

func (m *mockLogger) Debugf(msg string, kv ...any) {}
func (m *mockLogger) Infof(msg string, kv ...any)  {}
func (m *mockLogger) Warnf(msg string, kv ...any)  {}
func (m *mockLogger) Errorf(msg string, kv ...any) {}
func (m *mockLogger) With(kv ...any) core.Logger   { return m }

// mockReasoner is a mock reasoner for testing.
type mockReasoner struct {
	output map[string]any
	err    error
}

func (m *mockReasoner) Execute(ctx context.Context, req core.ReasonerRequest) (core.ReasonerResponse, error) {
	if m.err != nil {
		return core.ReasonerResponse{}, m.err
	}
	return core.ReasonerResponse{Output: m.output}, nil
}

func TestSymbolicPlanner_Plan_Success(t *testing.T) {
	// Create a mock reasoner that returns a valid plan
	params := map[string]any{"query": "test"}
	paramsJSON, _ := json.Marshal(params)

	mockOutput := map[string]any{
		"plan_step_0":   "0",
		"plan_tool_0":   "retriever",
		"plan_params_0": string(paramsJSON),
		"plan_reason_0": "Retrieve relevant documents",
		"plan_step_1":   "1",
		"plan_tool_1":   "llm",
		"plan_params_1": string(paramsJSON),
		"plan_reason_1": "Generate answer",
	}

	planner := &SymbolicPlanner{
		log:      &mockLogger{},
		reasoner: &mockReasoner{output: mockOutput},
	}

	query := core.Query{
		Text: "What is the capital of France?",
		Meta: map[string]any{"source": "test"},
	}

	plan, err := planner.Plan(context.Background(), query)
	require.NoError(t, err)
	require.Len(t, plan.Steps, 2)

	// Check first step
	assert.Equal(t, "retriever", plan.Steps[0].Tool)
	assert.Equal(t, "Retrieve relevant documents", plan.Steps[0].Reason)
	assert.NotNil(t, plan.Steps[0].Params)

	// Check second step
	assert.Equal(t, "llm", plan.Steps[1].Tool)
	assert.Equal(t, "Generate answer", plan.Steps[1].Reason)
	assert.NotNil(t, plan.Steps[1].Params)
}

func TestSymbolicPlanner_Plan_EmptyOutput(t *testing.T) {
	planner := &SymbolicPlanner{
		log:      &mockLogger{},
		reasoner: &mockReasoner{output: map[string]any{}},
	}

	query := core.Query{Text: "test query"}

	_, err := planner.Plan(context.Background(), query)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no plan steps")
}

func TestSymbolicPlanner_Plan_ReasonerError(t *testing.T) {
	planner := &SymbolicPlanner{
		log:      &mockLogger{},
		reasoner: &mockReasoner{err: assert.AnError},
	}

	query := core.Query{Text: "test query"}

	_, err := planner.Plan(context.Background(), query)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reasoner execution failed")
}

func TestSymbolicPlanner_ParseSteps_WithMapParams(t *testing.T) {
	planner := &SymbolicPlanner{log: &mockLogger{}}

	output := map[string]any{
		"plan_step_0":   "0",
		"plan_tool_0":   "tool1",
		"plan_params_0": map[string]any{"key": "value"},
		"plan_reason_0": "reason1",
	}

	steps, err := planner.parseSteps(output)
	require.NoError(t, err)
	require.Len(t, steps, 1)

	assert.Equal(t, "tool1", steps[0].Tool)
	assert.Equal(t, "reason1", steps[0].Reason)
	assert.Equal(t, "value", steps[0].Params["key"])
}

func TestSymbolicPlanner_ParseSteps_MultipleSteps(t *testing.T) {
	planner := &SymbolicPlanner{log: &mockLogger{}}

	output := map[string]any{
		"plan_tool_0":   "tool1",
		"plan_reason_0": "reason1",
		"plan_tool_2":   "tool3",
		"plan_reason_2": "reason3",
		"plan_tool_1":   "tool2",
		"plan_reason_1": "reason2",
	}

	steps, err := planner.parseSteps(output)
	require.NoError(t, err)
	require.Len(t, steps, 3)

	// Verify steps are sorted by order
	assert.Equal(t, "tool1", steps[0].Tool)
	assert.Equal(t, "tool2", steps[1].Tool)
	assert.Equal(t, "tool3", steps[2].Tool)
}

func TestSymbolicPlanner_ParseSteps_MissingTool(t *testing.T) {
	planner := &SymbolicPlanner{log: &mockLogger{}}

	output := map[string]any{
		"plan_step_0":   "0",
		"plan_reason_0": "reason1",
		// Missing plan_tool_0
	}

	steps, err := planner.parseSteps(output)
	require.NoError(t, err)
	// Step should be skipped due to missing tool
	assert.Len(t, steps, 0)
}

func TestSymbolicPlanner_ParseSteps_InvalidJSON(t *testing.T) {
	planner := &SymbolicPlanner{log: &mockLogger{}}

	output := map[string]any{
		"plan_tool_0":   "tool1",
		"plan_params_0": "invalid json {",
		"plan_reason_0": "reason1",
	}

	steps, err := planner.parseSteps(output)
	require.NoError(t, err)
	require.Len(t, steps, 1)

	// Invalid JSON should be wrapped in raw params
	assert.Equal(t, "tool1", steps[0].Tool)
	assert.NotNil(t, steps[0].Params)
	assert.Equal(t, "invalid json {", steps[0].Params["raw"])
}
