package pipeline

import (
	"errors"
	"strings"
	"testing"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/logger"
)

type mockRuleSet struct {
	result core.RuleResult
	err    error
}

func (m *mockRuleSet) Evaluate(stage core.Stage, q core.Query, a *core.Answer) (core.RuleResult, error) {
	if m.err != nil {
		return core.RuleResult{}, m.err
	}
	return m.result, nil
}

func TestPreRulesStage_Execute(t *testing.T) {
	testLogger := logger.NewStdLogger()

	t.Run("pre-rules allowed", func(t *testing.T) {
		rules := &mockRuleSet{result: core.RuleResult{Allowed: true}}
		stage := &PreRulesStage{RuleSet: rules, Logger: testLogger}
		p := &PipelineContext{}
		err := stage.Execute(p)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("pre-rules denied", func(t *testing.T) {
		rules := &mockRuleSet{result: core.RuleResult{Allowed: false, Reason: "denied"}}
		stage := &PreRulesStage{RuleSet: rules, Logger: testLogger}
		p := &PipelineContext{}
		err := stage.Execute(p)
		if !errors.Is(err, core.ErrDenied) {
			t.Errorf("expected error %v, got %v", core.ErrDenied, err)
		}
	})

	t.Run("pre-rules mutate", func(t *testing.T) {
		rules := &mockRuleSet{
			result: core.RuleResult{
				Allowed: true,
				Mutate: func(q *core.Query, a *core.Answer) {
					q.Text = "mutated"
				},
			},
		}
		stage := &PreRulesStage{RuleSet: rules, Logger: testLogger}
		p := &PipelineContext{Query: core.Query{Text: "original"}}
		err := stage.Execute(p)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if p.Query.Text != "mutated" {
			t.Errorf("expected query to be 'mutated', got '%s'", p.Query.Text)
		}
	})

	t.Run("pre-rules returns error", func(t *testing.T) {
		testErr := errors.New("rules error")
		rules := &mockRuleSet{err: testErr}
		stage := &PreRulesStage{RuleSet: rules, Logger: testLogger}
		p := &PipelineContext{}
		err := stage.Execute(p)
		if !strings.Contains(err.Error(), testErr.Error()) {
			t.Errorf("expected error to contain '%v', got '%v'", testErr, err)
		}
	})

	t.Run("nil ruleset is a no-op", func(t *testing.T) {
		stage := &PreRulesStage{RuleSet: nil, Logger: testLogger}
		p := &PipelineContext{}
		err := stage.Execute(p)
		if err != nil {
			t.Errorf("unexpected error for nil ruleset: %v", err)
		}
	})
}

func TestPostRulesStage_Execute(t *testing.T) {
	testLogger := logger.NewStdLogger()

	t.Run("post-rules mutate", func(t *testing.T) {
		rules := &mockRuleSet{
			result: core.RuleResult{
				Allowed: true,
				Mutate: func(q *core.Query, a *core.Answer) {
					a.Text = "mutated answer"
				},
			},
		}
		stage := &PostRulesStage{RuleSet: rules, Logger: testLogger}
		p := &PipelineContext{Answer: core.Answer{Text: "original"}}
		err := stage.Execute(p)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if p.Answer.Text != "mutated answer" {
			t.Errorf("expected answer to be 'mutated answer', got '%s'", p.Answer.Text)
		}
	})
}
