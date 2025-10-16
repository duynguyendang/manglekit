package pipeline

import (
	"errors"
	"testing"
)

// mockStage is a helper for testing the runner.
type mockStage struct {
	name    string
	err     error
	wasRun  bool
	mutator func(p *PipelineContext)
}

func (s *mockStage) Name() string {
	return s.name
}

func (s *mockStage) Execute(p *PipelineContext) error {
	s.wasRun = true
	if s.mutator != nil {
		s.mutator(p)
	}
	return s.err
}

func TestRunner_Run(t *testing.T) {
	t.Run("should run all stages in order", func(t *testing.T) {
		stage1 := &mockStage{name: "one", mutator: func(p *PipelineContext) {
			p.Response += "one;"
		}}
		stage2 := &mockStage{name: "two", mutator: func(p *PipelineContext) {
			p.Response += "two;"
		}}
		stage3 := &mockStage{name: "three", mutator: func(p *PipelineContext) {
			p.Response += "three;"
		}}

		runner := &Runner{}
		runner.Add(stage1)
		runner.Add(stage2)
		runner.Add(stage3)

		p := &PipelineContext{}
		err := runner.Run(p)

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if !stage1.wasRun || !stage2.wasRun || !stage3.wasRun {
			t.Error("expected all stages to run")
		}
		expectedOrder := "one;two;three;"
		if p.Response != expectedOrder {
			t.Errorf("expected stages to run in order, got %q, want %q", p.Response, expectedOrder)
		}
	})

	t.Run("should short-circuit on first error", func(t *testing.T) {
		testErr := errors.New("test error")
		stage1 := &mockStage{name: "one"}
		stage2 := &mockStage{name: "two", err: testErr}
		stage3 := &mockStage{name: "three"}

		runner := &Runner{}
		runner.Add(stage1)
		runner.Add(stage2)
		runner.Add(stage3)

		p := &PipelineContext{}
		err := runner.Run(p)

		if !errors.Is(err, testErr) {
			t.Errorf("expected error %v, got %v", testErr, err)
		}
		if !stage1.wasRun {
			t.Error("expected stage1 to run")
		}
		if !stage2.wasRun {
			t.Error("expected stage2 to run (and fail)")
		}
		if stage3.wasRun {
			t.Error("expected stage3 not to run")
		}
		if !errors.Is(p.Err, testErr) {
			t.Errorf("expected context error to be %v, got %v", testErr, p.Err)
		}
	})

	t.Run("should not add nil stages", func(t *testing.T) {
		runner := &Runner{}
		runner.Add(nil)
		if len(runner.stages) != 0 {
			t.Errorf("expected runner to have 0 stages, got %d", len(runner.stages))
		}
	})
}
