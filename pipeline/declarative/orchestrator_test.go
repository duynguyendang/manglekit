package declarative

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/providers/mock"
	"github.com/google/mangle/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockFlowController is a simplified mock for testing the declarative orchestrator.
type mockFlowController struct {
	mu           sync.Mutex
	decls        map[ast.PredicateSym]ast.Decl
	clauses      []ast.Clause
	transformers map[ast.PredicateSym]string
	evalResult   core.RuleResult
	queryResults map[string][]map[string]any
	err          error // Generic error for any method
}

func newMockFlowController() *mockFlowController {
	return &mockFlowController{
		queryResults: map[string][]map[string]any{
			`flow_stage("test", Order, StageName).`: {
				{"Order": "1", "StageName": "one"},
				{"Order": "2", "StageName": "two"},
			},
			`stage_tool(StageName, ToolName).`: {
				{"StageName": "one", "ToolName": "toolOne"},
				{"StageName": "two", "ToolName": "toolTwo"},
			},
		},
	}
}

func (m *mockFlowController) GetDeclarations(context.Context, string) (map[ast.PredicateSym]ast.Decl, []ast.Clause, error) {
	if m.err != nil {
		return nil, nil, m.err
	}
	return m.decls, m.clauses, nil
}
func (m *mockFlowController) GetTransformer(context.Context, ast.PredicateSym) (string, bool, error) {
	return "", false, nil
}
func (m *mockFlowController) Close() error { return nil }

func (m *mockFlowController) Evaluate(stage core.Stage, q core.Query, a *core.Answer) (core.RuleResult, error) {
	if m.err != nil {
		return core.RuleResult{}, m.err
	}
	if q.Text == "skip" {
		return core.RuleResult{Allowed: true, SkippedStages: map[string]bool{"two": true}}, nil
	}
	return core.RuleResult{Allowed: true}, nil
}

func (m *mockFlowController) Query(ctx context.Context, query string, onSolution func(map[string]any) error) error {
	if m.err != nil {
		return m.err
	}
	if results, ok := m.queryResults[query]; ok {
		for _, res := range results {
			if err := onSolution(res); err != nil {
				return err
			}
		}
	}
	return nil
}

func TestNew(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		fc          core.FlowController
		tools       map[string]any
		expectErr   bool
		errContains string
	}{
		{
			name:  "Happy path",
			fc:    newMockFlowController(),
			tools: make(map[string]any),
		},
		{
			name:      "Nil flow controller",
			fc:        nil,
			tools:     make(map[string]any),
			expectErr: true,
		},
		{
			name:      "Nil tools map",
			fc:        newMockFlowController(),
			tools:     nil,
			expectErr: true,
		},
		{
			name: "Flow controller fails to load stages",
			fc: &mockFlowController{
				err: errors.New("stage loading failed"),
			},
			tools:       make(map[string]any),
			expectErr:   true,
			errContains: "stage loading failed",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			orch, err := New(tc.fc, tc.tools, "test", core.Observability{}, nil)
			if tc.expectErr {
				require.Error(t, err)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
				assert.Nil(t, orch)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, orch)
			}
		})
	}
}

func TestDeclarativeOrchestrator_Run(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		query        core.Query
		tools        map[string]any
		fc           *mockFlowController
		validateFunc func(t *testing.T, answer core.Answer, tools map[string]any)
		expectErr    bool
		errContains  string
	}{
		{
			name:  "Successful run with two stages",
			query: core.Query{Text: "run"},
			tools: map[string]any{
				"toolOne": &mock.Tool{Fn: func(p mock.Params) (mock.Object, error) {
					p["__called"] = true
					return mock.Object{"a": "foo"}, nil
				}},
				"toolTwo": &mock.Tool{Fn: func(p mock.Params) (mock.Object, error) {
					p["__called"] = true
					return mock.Object{"b": "bar"}, nil
				}},
			},
			validateFunc: func(t *testing.T, answer core.Answer, tools map[string]any) {
				params1 := tools["toolOne"].(*mock.Tool).GetLastParams()
				require.NotNil(t, params1)
				assert.True(t, params1["__called"].(bool))

				params2 := tools["toolTwo"].(*mock.Tool).GetLastParams()
				require.NotNil(t, params2)
				assert.True(t, params2["__called"].(bool))

				assert.Equal(t, "foo", answer.Meta["a"])
				assert.Equal(t, "bar", answer.Meta["b"])
			},
		},
		{
			name:  "Skip a stage",
			query: core.Query{Text: "skip"},
			tools: map[string]any{
				"toolOne": &mock.Tool{Fn: func(p mock.Params) (mock.Object, error) {
					p["__called"] = true
					return nil, nil
				}},
				"toolTwo": &mock.Tool{Fn: func(p mock.Params) (mock.Object, error) {
					p["__called"] = true
					return nil, nil
				}},
			},
			validateFunc: func(t *testing.T, answer core.Answer, tools map[string]any) {
				params1 := tools["toolOne"].(*mock.Tool).GetLastParams()
				require.NotNil(t, params1)
				assert.True(t, params1["__called"].(bool))
				// toolTwo is in stage "two", which is skipped.
				params2 := tools["toolTwo"].(*mock.Tool).GetLastParams()
				assert.Empty(t, params2, "Expected last params for skipped tool to be empty")
			},
		},
		{
			name:  "Tool returns an error",
			query: core.Query{Text: "run"},
			tools: map[string]any{
				"toolOne": &mock.Tool{Fn: func(p mock.Params) (mock.Object, error) {
					return nil, errors.New("tool failed")
				}},
				"toolTwo": &mock.Tool{},
			},
			expectErr:   true,
			errContains: "tool failed",
		},
		{
			name: "Dependency inference for tool parameters",
			query: core.Query{
				Meta: map[string]any{"initial": "value"},
			},
			fc: &mockFlowController{
				queryResults: map[string][]map[string]any{
					`flow_stage("test", Order, StageName).`: {{"Order": "1", "StageName": "one"}},
					`stage_tool(StageName, ToolName).`:     {{"StageName": "one", "ToolName": "consumer"}},
				},
			},
			tools: map[string]any{
				"consumer": &mock.Tool{Fn: func(p mock.Params) (mock.Object, error) {
					assert.Equal(t, "value", p["initial"])
					return nil, nil
				}},
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fc := tc.fc
			if fc == nil {
				fc = newMockFlowController()
			}

			orch, err := New(fc, tc.tools, "test", core.Observability{}, nil)
			if err != nil && tc.name == "Flow controller fails to load stages" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
				return
			}
			require.NoError(t, err)

			answer, err := orch.Run(context.Background(), tc.query)

			if tc.expectErr {
				require.Error(t, err)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
			} else {
				require.NoError(t, err)
				if tc.validateFunc != nil {
					tc.validateFunc(t, answer, tc.tools)
				}
			}
		})
	}
}
