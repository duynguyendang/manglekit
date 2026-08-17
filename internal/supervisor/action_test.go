package supervisor

import (
	"context"
	"errors"
	"iter"
	"testing"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/core/domain"
)

type mockAction struct {
	executeResult domain.Envelope
	executeErr    error
	metadata      core.ActionMetadata
}

func (m *mockAction) Execute(ctx context.Context, input domain.Envelope) (domain.Envelope, error) {
	return m.executeResult, m.executeErr
}

type mockReasoningPortForSup struct {
	verifyAtomsResult *domain.AuditResult
	verifyAtomsErr    error
}

func (m *mockReasoningPortForSup) Verify(ctx context.Context, subject interface{}, genome []domain.DomainGene) (*domain.AuditResult, error) {
	return nil, nil
}

func (m *mockReasoningPortForSup) VerifyAtoms(ctx context.Context, atoms []domain.Atom, genome []domain.DomainGene) (*domain.AuditResult, error) {
	return m.verifyAtomsResult, m.verifyAtomsErr
}

func (m *mockReasoningPortForSup) Query(ctx context.Context, query string, genome []domain.DomainGene) ([]domain.Atom, error) {
	return nil, nil
}

type mockGenePoolPort struct {
	activeGenes []domain.DomainGene
}

func (m *mockGenePoolPort) ActiveGenes(ctx context.Context, intent domain.IntentStr) iter.Seq[*domain.DomainGene] {
	return func(yield func(*domain.DomainGene) bool) {
		for _, g := range m.activeGenes {
			g := g
			yield(&g)
		}
	}
}

func (m *mockGenePoolPort) Reload(ctx context.Context) error {
	return nil
}

type mockSupervisorAction struct {
	executeInput  any
	executeResult domain.Envelope
	executeErr    error
}

func (m *mockSupervisorAction) Execute(ctx context.Context, input domain.Envelope) (domain.Envelope, error) {
	m.executeInput = input.Payload
	return m.executeResult, m.executeErr
}

func TestSupervisedAction_ExecuteInternal_Pass(t *testing.T) {
	inner := &mockAction{
		executeResult: domain.Envelope{Payload: "result"},
	}
	verifier := &mockReasoningPortForSup{
		verifyAtomsResult: &domain.AuditResult{Pass: true},
	}
	genePool := &mockGenePoolPort{
		activeGenes: []domain.DomainGene{
			{Name: "test-gene", Tier: domain.Tier0Kernel},
		},
	}

	supervised := New(inner, verifier, genePool)

	env := core.Envelope{Payload: "test-input"}
	result, err := supervised.ExecuteInternal(context.Background(), "test-intent", env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Payload != "result" {
		t.Errorf("expected payload 'result', got %v", result.Payload)
	}
}

func TestSupervisedAction_ExecuteInternal_VerifierFailTier0(t *testing.T) {
	inner := &mockAction{
		executeResult: domain.Envelope{Payload: "result"},
	}
	verifier := &mockReasoningPortForSup{
		verifyAtomsResult: &domain.AuditResult{
			Pass:          false,
			ViolationTier: domain.Tier0Kernel,
			ConflictPath:  "test-rule",
		},
	}
	genePool := &mockGenePoolPort{
		activeGenes: []domain.DomainGene{
			{Name: "test-gene", Tier: domain.Tier0Kernel},
		},
	}

	supervised := New(inner, verifier, genePool)

	env := core.Envelope{Payload: "test-input"}
	_, err := supervised.ExecuteInternal(context.Background(), "test-intent", env)
	if err == nil {
		t.Fatal("expected error for Tier0 violation")
	}
	if !errors.Is(err, core.ErrPolicyViolation) {
		t.Errorf("expected ErrPolicyViolation, got %v", err)
	}
}

func TestSupervisedAction_ExecuteInternal_VerifierFailTier1(t *testing.T) {
	inner := &mockAction{
		executeResult: domain.Envelope{Payload: "result"},
	}
	verifier := &mockReasoningPortForSup{
		verifyAtomsResult: &domain.AuditResult{
			Pass:          false,
			ViolationTier: domain.Tier1Admin,
			ConflictPath:  "admin-rule",
		},
	}
	genePool := &mockGenePoolPort{
		activeGenes: []domain.DomainGene{
			{Name: "test-gene", Tier: domain.Tier0Kernel},
		},
	}

	supervised := New(inner, verifier, genePool)

	env := core.Envelope{Payload: "test-input"}
	_, err := supervised.ExecuteInternal(context.Background(), "test-intent", env)
	if err == nil {
		t.Fatal("expected error for Tier1 violation")
	}
	if !errors.Is(err, core.ErrPolicyViolation) {
		t.Errorf("expected ErrPolicyViolation, got %v", err)
	}
}

func TestSupervisedAction_ExecuteInternal_Tier2AllowsPass(t *testing.T) {
	inner := &mockAction{
		executeResult: domain.Envelope{Payload: "result"},
	}
	verifier := &mockReasoningPortForSup{
		verifyAtomsResult: &domain.AuditResult{
			Pass:          false,
			ViolationTier: domain.Tier2AI,
			ConflictPath:  "playbook-rule",
		},
	}
	genePool := &mockGenePoolPort{
		activeGenes: []domain.DomainGene{
			{Name: "test-gene", Tier: domain.Tier0Kernel},
		},
	}

	supervised := New(inner, verifier, genePool)

	env := core.Envelope{Payload: "test-input"}
	result, err := supervised.ExecuteInternal(context.Background(), "test-intent", env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Payload != "result" {
		t.Errorf("expected payload 'result', got %v", result.Payload)
	}
	// Tier2 violations don't block execution, no violations added for soft rules
}

func TestSupervisedAction_ExecuteInternal_InnerError(t *testing.T) {
	innerErr := errors.New("inner execution failed")
	inner := &mockAction{
		executeErr: innerErr,
	}
	verifier := &mockReasoningPortForSup{
		verifyAtomsResult: &domain.AuditResult{Pass: true},
	}
	genePool := &mockGenePoolPort{
		activeGenes: []domain.DomainGene{
			{Name: "test-gene", Tier: domain.Tier0Kernel},
		},
	}

	supervised := New(inner, verifier, genePool)

	env := core.Envelope{Payload: "test-input"}
	_, err := supervised.ExecuteInternal(context.Background(), "test-intent", env)
	if err == nil {
		t.Fatal("expected error from inner")
	}
}

func TestSupervisedAction_ExecuteInternal_VerifierError(t *testing.T) {
	inner := &mockAction{
		executeResult: domain.Envelope{Payload: "result"},
	}
	verifier := &mockReasoningPortForSup{
		verifyAtomsErr: errors.New("verifier error"),
	}
	genePool := &mockGenePoolPort{
		activeGenes: []domain.DomainGene{
			{Name: "test-gene", Tier: domain.Tier0Kernel},
		},
	}

	supervised := New(inner, verifier, genePool)

	env := core.Envelope{Payload: "test-input"}
	_, err := supervised.ExecuteInternal(context.Background(), "test-intent", env)
	if err == nil {
		t.Fatal("expected error from verifier")
	}
}

func TestSupervisedAction_FlattenToQuads_WithTags(t *testing.T) {
	inner := &mockAction{}
	verifier := &mockReasoningPortForSup{
		verifyAtomsResult: &domain.AuditResult{Pass: true},
	}
	genePool := &mockGenePoolPort{
		activeGenes: []domain.DomainGene{
			{Name: "test-gene", Tier: domain.Tier0Kernel},
		},
	}

	supervised := New(inner, verifier, genePool)

	type testStruct struct {
		Name string `mangle:"name"`
		Age  int    `mangle:"age"`
	}

	quads := supervised.flattenToQuads("TestSubject", testStruct{Name: "Alice", Age: 30})
	if len(quads) != 2 {
		t.Errorf("expected 2 quads, got %d", len(quads))
	}
}

func TestSupervisedAction_FlattenToQuads_NoTags(t *testing.T) {
	inner := &mockAction{}
	verifier := &mockReasoningPortForSup{
		verifyAtomsResult: &domain.AuditResult{Pass: true},
	}
	genePool := &mockGenePoolPort{
		activeGenes: []domain.DomainGene{
			{Name: "test-gene", Tier: domain.Tier0Kernel},
		},
	}

	supervised := New(inner, verifier, genePool)

	type testStruct struct {
		Name string
		Age  int
	}

	quads := supervised.flattenToQuads("TestSubject", testStruct{Name: "Alice", Age: 30})
	if len(quads) != 0 {
		t.Errorf("expected 0 quads for untagged struct, got %d", len(quads))
	}
}

func TestSupervisedAction_FlattenToQuads_NonStruct(t *testing.T) {
	inner := &mockAction{}
	verifier := &mockReasoningPortForSup{
		verifyAtomsResult: &domain.AuditResult{Pass: true},
	}
	genePool := &mockGenePoolPort{
		activeGenes: []domain.DomainGene{
			{Name: "test-gene", Tier: domain.Tier0Kernel},
		},
	}

	supervised := New(inner, verifier, genePool)

	quads := supervised.flattenToQuads("TestSubject", 123)
	if len(quads) != 0 {
		t.Errorf("expected 0 quads for non-struct, got %d", len(quads))
	}

	quads = supervised.flattenToQuads("TestSubject", "string")
	if len(quads) != 0 {
		t.Errorf("expected 0 quads for string, got %d", len(quads))
	}
}

func TestSupervisedAction_FlattenToQuads_Pointer(t *testing.T) {
	inner := &mockAction{}
	verifier := &mockReasoningPortForSup{
		verifyAtomsResult: &domain.AuditResult{Pass: true},
	}
	genePool := &mockGenePoolPort{
		activeGenes: []domain.DomainGene{
			{Name: "test-gene", Tier: domain.Tier0Kernel},
		},
	}

	supervised := New(inner, verifier, genePool)

	type testStruct struct {
		Name string `mangle:"name"`
	}

	s := &testStruct{Name: "Bob"}
	quads := supervised.flattenToQuads("TestSubject", s)
	if len(quads) != 1 {
		t.Errorf("expected 1 quad for pointer to struct, got %d", len(quads))
	}
}

// multiCallReasoningPort returns different results per VerifyAtoms call,
// so tests can distinguish the pre-check (call 1) from the post-check
// (call 2, Reflect).
type multiCallReasoningPort struct {
	results []*domain.AuditResult
	errs    []error
	calls   int
}

func (m *multiCallReasoningPort) Verify(ctx context.Context, subject interface{}, genome []domain.DomainGene) (*domain.AuditResult, error) {
	return nil, nil
}

func (m *multiCallReasoningPort) VerifyAtoms(ctx context.Context, atoms []domain.Atom, genome []domain.DomainGene) (*domain.AuditResult, error) {
	i := m.calls
	m.calls++
	if i >= len(m.results) {
		return &domain.AuditResult{Pass: true}, nil
	}
	return m.results[i], m.errs[i]
}

func (m *multiCallReasoningPort) Query(ctx context.Context, query string, genome []domain.DomainGene) ([]domain.Atom, error) {
	return nil, nil
}

// postCheckTestEnv builds a context carrying the pre-check context so
// ExecuteInternal runs the post-check (Reflect) path.
func postCheckTestEnv(ctx context.Context) context.Context {
	return withPreCheckContext(ctx, &preCheckContext{
		actionName: "test-action",
		entityID:   core.EntityInput,
	})
}

func TestSupervisedAction_PostCheck_VerifierError_FailClosed(t *testing.T) {
	inner := &mockAction{
		executeResult: domain.Envelope{Payload: "result"},
	}
	verifier := &multiCallReasoningPort{
		results: []*domain.AuditResult{
			{Pass: true}, // pre-check passes
			{Pass: false, ViolationTier: domain.Tier0Kernel}, // unused; error below wins
		},
		errs: []error{
			nil,
			errors.New("verifier exploded during Reflect"),
		},
	}
	genePool := &mockGenePoolPort{activeGenes: []domain.DomainGene{{Name: "g", Tier: domain.Tier0Kernel}}}

	supervised := New(inner, verifier, genePool)

	_, err := supervised.ExecuteInternal(postCheckTestEnv(context.Background()), "test-intent", core.Envelope{Payload: "test-input"})
	if err == nil {
		t.Fatal("expected error when post-check verifier errors (fail-closed)")
	}
	if !errors.Is(err, core.ErrSupervisorFailure) {
		t.Errorf("expected ErrSupervisorFailure, got %v", err)
	}
	if verifier.calls != 2 {
		t.Errorf("expected pre-check + post-check = 2 verifier calls, got %d", verifier.calls)
	}
}

func TestSupervisedAction_PostCheck_Violation_Blocks(t *testing.T) {
	inner := &mockAction{
		executeResult: domain.Envelope{Payload: "result"},
	}
	verifier := &multiCallReasoningPort{
		results: []*domain.AuditResult{
			{Pass: true}, // pre-check passes
			{Pass: false, ViolationTier: domain.Tier1Admin, ConflictPath: "post-rule"}, // post-check fails
		},
		errs: []error{nil, nil},
	}
	genePool := &mockGenePoolPort{activeGenes: []domain.DomainGene{{Name: "g", Tier: domain.Tier0Kernel}}}

	supervised := New(inner, verifier, genePool)

	_, err := supervised.ExecuteInternal(postCheckTestEnv(context.Background()), "test-intent", core.Envelope{Payload: "test-input"})
	if err == nil {
		t.Fatal("expected error when post-check fails")
	}
	if !errors.Is(err, core.ErrPolicyViolation) {
		t.Errorf("expected ErrPolicyViolation, got %v", err)
	}
}

func TestSupervisedAction_PostCheck_Pass_StillSucceeds(t *testing.T) {
	inner := &mockAction{
		executeResult: domain.Envelope{Payload: "result"},
	}
	verifier := &multiCallReasoningPort{
		results: []*domain.AuditResult{
			{Pass: true},
			{Pass: true},
		},
		errs: []error{nil, nil},
	}
	genePool := &mockGenePoolPort{activeGenes: []domain.DomainGene{{Name: "g", Tier: domain.Tier0Kernel}}}

	supervised := New(inner, verifier, genePool)

	result, err := supervised.ExecuteInternal(postCheckTestEnv(context.Background()), "test-intent", core.Envelope{Payload: "test-input"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Payload != "result" {
		t.Errorf("expected payload 'result', got %v", result.Payload)
	}
}
