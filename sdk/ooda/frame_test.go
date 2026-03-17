package ooda

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/duynguyendang/manglekit/core"
)

type mockMemory struct {
	atoms     []Atom
	commitErr error
}

func (m *mockMemory) Recall(ctx context.Context, input string) ([]Atom, error) {
	return m.atoms, nil
}

func (m *mockMemory) Commit(ctx context.Context, frame *CognitiveFrame) error {
	return m.commitErr
}

func (m *mockMemory) Store(ctx context.Context, atom Atom) error {
	m.atoms = append(m.atoms, atom)
	return nil
}

func (m *mockMemory) Query(ctx context.Context, predicate string) ([]Atom, error) {
	var result []Atom
	for _, atom := range m.atoms {
		if atom.Predicate == predicate {
			result = append(result, atom)
		}
	}
	return result, nil
}

type mockBrain struct {
	decision *core.Decision
	err      error
}

func (b *mockBrain) Evaluate(ctx context.Context, frame *CognitiveFrame) (*core.Decision, error) {
	if b.err != nil {
		return nil, b.err
	}
	return b.decision, nil
}

func (b *mockBrain) Verify(ctx context.Context, frame *CognitiveFrame) (*core.AuditTrail, error) {
	return &core.AuditTrail{
		MatchedRules: []core.RuleInference{
			{
				RuleName: "verify_pass",
				Tier:     core.TierT1_Governance,
			},
		},
	}, nil
}

func (b *mockBrain) LoadPolicy(ctx context.Context, rules string) error {
	return nil
}

type mockExecutor struct {
	result      any
	executeErr  error
	rollbackErr error
}

func (e *mockExecutor) Execute(ctx context.Context, frame *CognitiveFrame, decision *core.Decision) (any, error) {
	if e.executeErr != nil {
		return nil, e.executeErr
	}
	return e.result, nil
}

func (e *mockExecutor) Rollback(ctx context.Context, frame *CognitiveFrame, result any) error {
	return e.rollbackErr
}

func TestBuilder(t *testing.T) {
	frame := NewBuilder().
		WithInput("test input").
		WithIntent(IntentStr("test intent")).
		WithTaskType(TaskTypeGeneration).
		WithMemory(&mockMemory{}).
		WithBrain(&mockBrain{}).
		WithExecutor(&mockExecutor{result: "success"}).
		WithMaxRetries(5).
		WithTimeout(10 * time.Minute).
		WithTraceID("test-trace").
		Build()

	if frame.Input != "test input" {
		t.Errorf("expected input 'test input', got %s", frame.Input)
	}

	if frame.Intent != "test intent" {
		t.Errorf("expected intent 'test intent', got %s", frame.Intent)
	}

	if frame.Memory == nil {
		t.Error("expected memory to be set")
	}

	if frame.Brain == nil {
		t.Error("expected brain to be set")
	}

	if frame.Executor == nil {
		t.Error("expected executor to be set")
	}

	if frame.MaxRetries != 5 {
		t.Errorf("expected max retries 5, got %d", frame.MaxRetries)
	}

	if frame.Timeout != 10*time.Minute {
		t.Errorf("expected timeout 10m, got %v", frame.Timeout)
	}

	if frame.TraceID != "test-trace" {
		t.Errorf("expected trace ID 'test-trace', got %s", frame.TraceID)
	}
}

func TestOodaLoop_AllPhases(t *testing.T) {
	ctx := context.Background()

	memory := &mockMemory{
		atoms: []Atom{
			{Predicate: "context_fact", Subject: "test", Object: "value", Weight: 0.8},
		},
	}

	brain := &mockBrain{
		decision: &core.Decision{
			Outcome: core.DecisionProceed,
			AuditTrail: &core.AuditTrail{
				MatchedRules: []core.RuleInference{
					{RuleName: "allow_action", Tier: core.TierT1_Governance},
				},
			},
		},
	}

	executor := &mockExecutor{
		result: "action completed",
	}

	frame := NewBuilder().
		WithInput("test request").
		WithMemory(memory).
		WithBrain(brain).
		WithExecutor(executor).
		Build()

	resultFrame, err := Run(ctx, frame)
	if err != nil {
		t.Fatalf("OODA loop failed: %v", err)
	}

	// Verify phases executed
	if resultFrame.Phase != PhaseVerify {
		t.Errorf("expected final phase to be Verify, got %s", resultFrame.Phase)
	}

	// Verify context was hydrated
	if len(resultFrame.Context) < 2 { // raw_input + context_fact
		t.Errorf("expected context to be hydrated, got %d atoms", len(resultFrame.Context))
	}

	// Verify decision was made
	if resultFrame.Decision == nil {
		t.Error("expected decision to be set")
	}

	// Verify action result
	if resultFrame.ActionResult != "action completed" {
		t.Errorf("expected action result 'action completed', got %v", resultFrame.ActionResult)
	}

	// Verify audit trail
	if resultFrame.AuditTrail == nil {
		t.Error("expected audit trail to be set")
	}

	// Verify phase durations
	durations := resultFrame.GetPhaseDurations()
	if len(durations) == 0 {
		t.Error("expected phase durations to be recorded")
	}

	totalDuration := resultFrame.TotalDuration()
	if totalDuration <= 0 {
		t.Error("expected positive total duration")
	}

	t.Logf("OODA loop completed in %v", totalDuration)
	t.Logf("Audit summary: %s", resultFrame.GetAuditSummary())
}

func TestOodaLoop_Retry(t *testing.T) {
	ctx := context.Background()

	attemptCount := 0

	// Custom executor that fails first 2 times
	failingExecutor := &mockExecutorWithRetry{
		failUntil: 2,
		attempts:  &attemptCount,
		result:    "success after retry",
	}

	brain := &mockBrain{
		decision: &core.Decision{Outcome: core.DecisionProceed},
	}

	frame := NewBuilder().
		WithInput("test retry").
		WithBrain(brain).
		WithExecutor(failingExecutor).
		WithMaxRetries(3).
		Build()

	resultFrame, err := Run(ctx, frame)
	if err != nil {
		t.Fatalf("OODA loop failed: %v", err)
	}

	if resultFrame.RetryCount != 2 {
		t.Errorf("expected 2 retries, got %d", resultFrame.RetryCount)
	}

	if resultFrame.ActionResult != "success after retry" {
		t.Errorf("expected final result 'success after retry', got %v", resultFrame.ActionResult)
	}

	t.Logf("Succeeded after %d retries", attemptCount)
}

type mockExecutorWithRetry struct {
	failUntil int
	attempts  *int
	result    any
}

func (e *mockExecutorWithRetry) Execute(ctx context.Context, frame *CognitiveFrame, decision *core.Decision) (any, error) {
	*e.attempts++
	if *e.attempts <= e.failUntil {
		return nil, fmt.Errorf("intentional failure on attempt %d", *e.attempts)
	}
	return e.result, nil
}

func (e *mockExecutorWithRetry) Rollback(ctx context.Context, frame *CognitiveFrame, result any) error {
	return nil
}

func TestOodaLoop_NoBrain(t *testing.T) {
	ctx := context.Background()

	frame := NewBuilder().
		WithInput("test no brain").
		WithMemory(&mockMemory{}).
		WithExecutor(&mockExecutor{result: "result"}).
		Build()

	resultFrame, err := Run(ctx, frame)
	if err != nil {
		t.Fatalf("OODA loop failed: %v", err)
	}

	// When there's no brain, there's no decision, so no action is executed
	// This is expected behavior - the frame just goes through observe/orient/verify
	t.Logf("Final phase: %s", resultFrame.Phase)
	t.Logf("Action result: %v", resultFrame.ActionResult)
}

func TestOodaLoop_NoMemory(t *testing.T) {
	ctx := context.Background()

	frame := NewBuilder().
		WithInput("test no memory").
		WithBrain(&mockBrain{decision: &core.Decision{Outcome: core.DecisionProceed}}).
		WithExecutor(&mockExecutor{result: "result"}).
		Build()

	resultFrame, err := Run(ctx, frame)
	if err != nil {
		t.Fatalf("OODA loop failed: %v", err)
	}

	// Should still execute even without memory
	if resultFrame.ActionResult != "result" {
		t.Errorf("expected result 'result', got %v", resultFrame.ActionResult)
	}
}

func TestGetAuditSummary(t *testing.T) {
	frame := &CognitiveFrame{
		AuditTrail: &core.AuditTrail{
			MatchedRules: []core.RuleInference{
				{RuleName: "rule1", Tier: core.TierT1_Governance},
				{RuleName: "rule2", Tier: core.TierT2_Playbook},
			},
		},
	}

	summary := frame.GetAuditSummary()
	if summary == "" {
		t.Error("expected non-empty summary")
	}

	t.Logf("Audit Summary:\n%s", summary)
}

func TestGetPhaseDurations(t *testing.T) {
	frame := &CognitiveFrame{
		PhaseDurations: map[Phase]time.Duration{
			PhaseObserve: 10 * time.Millisecond,
			PhaseOrient:  20 * time.Millisecond,
			PhaseDecide:  30 * time.Millisecond,
			PhaseAct:     40 * time.Millisecond,
			PhaseVerify:  50 * time.Millisecond,
		},
	}

	durations := frame.GetPhaseDurations()
	if len(durations) != 5 {
		t.Errorf("expected 5 phase durations, got %d", len(durations))
	}

	total := frame.TotalDuration()
	if total != 150*time.Millisecond {
		t.Errorf("expected total 150ms, got %v", total)
	}
}

var _ = fmt.Printf // Import needed for fmt in test
