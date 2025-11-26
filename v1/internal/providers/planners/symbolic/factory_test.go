package symbolic_test

import (
	"context"
	"testing"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
	"github.com/duynguyendang/manglekit/internal/providers/planners/symbolic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockReasoner is a mock implementation of core.Reasoner for testing.
type mockReasoner struct{}

func (m *mockReasoner) Execute(ctx context.Context, req core.ReasonerRequest) (core.ReasonerResponse, error) {
	// Return a simple plan with one step
	output := map[string]any{
		"plan_tool_0":   "retriever",
		"plan_reason_0": "Retrieve documents",
		"plan_params_0": `{"query": "test"}`,
	}
	return core.ReasonerResponse{Output: output}, nil
}

func TestSymbolicPlanner_Integration_Success(t *testing.T) {
	// Test the factory directly with proper dependencies
	mockLog := &mockLogger{}
	deps := diapi.PlannerDeps{
		CoreDeps: diapi.CoreDeps{
			Obs: core.Observability{
				Logger: mockLog,
			},
		},
		Reasoners: map[string]core.Reasoner{
			"test-reasoner": &mockReasoner{},
		},
	}

	opts := &symbolic.Options{
		ReasonerName: "test-reasoner",
	}

	planner, err := symbolic.NewFactory(deps, opts)
	require.NoError(t, err, "factory should build planner successfully")
	require.NotNil(t, planner)

	// Test the planner works
	query := core.Query{Text: "test query"}
	plan, err := planner.Plan(context.Background(), query)
	require.NoError(t, err)
	require.Len(t, plan.Steps, 1)

	assert.Equal(t, "retriever", plan.Steps[0].Tool)
	assert.Equal(t, "Retrieve documents", plan.Steps[0].Reason)
}

// mockLogger for integration tests
type mockLogger struct{}

func (m *mockLogger) Debugf(msg string, kv ...any) {}
func (m *mockLogger) Infof(msg string, kv ...any)  {}
func (m *mockLogger) Warnf(msg string, kv ...any)  {}
func (m *mockLogger) Errorf(msg string, kv ...any) {}
func (m *mockLogger) With(kv ...any) core.Logger   { return m }

func TestSymbolicPlanner_Integration_MissingReasoner(t *testing.T) {
	mockLog := &mockLogger{}
	deps := diapi.PlannerDeps{
		CoreDeps: diapi.CoreDeps{
			Obs: core.Observability{
				Logger: mockLog,
			},
		},
		Reasoners: map[string]core.Reasoner{},
	}

	opts := &symbolic.Options{
		ReasonerName: "nonexistent-reasoner",
	}

	_, err := symbolic.NewFactory(deps, opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestSymbolicPlanner_Integration_MissingReasonerParam(t *testing.T) {
	mockLog := &mockLogger{}
	deps := diapi.PlannerDeps{
		CoreDeps: diapi.CoreDeps{
			Obs: core.Observability{
				Logger: mockLog,
			},
		},
		Reasoners: map[string]core.Reasoner{
			"test-reasoner": &mockReasoner{},
		},
	}

	opts := &symbolic.Options{
		ReasonerName: "", // Missing reasoner name
	}

	_, err := symbolic.NewFactory(deps, opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires a reasoner name")
}

func TestSymbolicPlanner_Options_ProviderMethods(t *testing.T) {
	opts := &symbolic.Options{ReasonerName: "test"}

	assert.Equal(t, "symbolic", opts.ProviderName())
	assert.Equal(t, core.KindPlanner, opts.ProviderKind())
	assert.Equal(t, opts, opts.GetProviderOptions())
}
