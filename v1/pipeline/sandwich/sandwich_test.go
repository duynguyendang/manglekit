package sandwich

import (
	"context"
	"testing"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
	"github.com/duynguyendang/manglekit/internal/logger"
	"github.com/duynguyendang/manglekit/internal/testproviders/mock"
	"github.com/google/mangle/ast"
	"github.com/stretchr/testify/assert"
)

// mockRuleSet is a mock implementation of core.RuleSet for testing.
type mockRuleSet struct {
	targetAction string
}

func (m *mockRuleSet) Evaluate(ctx context.Context, stage core.Stage, q core.Query, a *core.Answer) (core.RuleResult, error) {
	return core.RuleResult{Allowed: true}, nil
}

func (m *mockRuleSet) EvaluateFacts(ctx context.Context, stage core.Stage, facts []ast.Atom, a *core.Answer) (core.RuleResult, error) {
	if m.targetAction != "" {
		a.Meta["target_action"] = m.targetAction
	}
	return core.RuleResult{Allowed: true}, nil
}

func TestSandwichOrchestrator_SmartRouter(t *testing.T) {
	ctx := context.Background()

	defaultAction := &core.RetrieverAction{
		Retriever: mock.NewRetriever(map[string]string{"default": "default-action-result"}),
	}
	subAction := &core.RetrieverAction{
		Retriever: mock.NewRetriever(map[string]string{"sub": "sub-action-result"}),
	}

	testCases := []struct {
		name              string
		ruleSet           core.RuleSet
		query             core.Query
		expectedAction    string
		expectedDoc       string
		expectedQueryText string
	}{
		{
			name:              "Default action",
			ruleSet:           &mockRuleSet{},
			query:             core.Query{Text: "default"},
			expectedAction:    "default",
			expectedDoc:       "default-action-result",
			expectedQueryText: "default",
		},
		{
			name:              "Sub-action triggered",
			ruleSet:           &mockRuleSet{targetAction: "sub-action"},
			query:             core.Query{Text: "sub"},
			expectedAction:    "sub-action",
			expectedDoc:       "sub-action-result",
			expectedQueryText: "sub",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			deps := diapi.SandwichDeps{
				CoreDeps: diapi.CoreDeps{
					Obs: core.Observability{Logger: logger.NewStdLogger()},
				},
				Action:  defaultAction,
				RuleSet: tc.ruleSet,
				SubActions: map[string]core.Action{
					"sub-action": subAction,
				},
			}

			factory := NewFactory()
			built, err := factory.Build(ctx, deps, &Options{})
			assert.NoError(t, err)
			orchestrator := built.(core.Orchestrator)

			queryToRun := tc.query
			queryToRun.Text = tc.expectedQueryText

			answer, err := orchestrator.Execute(ctx, "session-123", queryToRun)
			assert.NoError(t, err)

			assert.Equal(t, tc.expectedAction, answer.Meta["executed_action"])
			if result, ok := answer.Meta["action_result"].(core.RetrieveResult); ok {
				if len(result.Docs) > 0 {
					assert.Equal(t, tc.expectedDoc, result.Docs[0].Text)
				} else {
					t.Error("No documents returned")
				}
			} else {
				t.Errorf("Unexpected result type: %T", answer.Meta["action_result"])
			}
		})
	}
}
