package orchestrator

import (
	"context"
	"errors"
	"iter"
	"os"
	"testing"

	"github.com/duynguyendang/manglekit/internal/core/domain"
	"github.com/duynguyendang/manglekit/internal/core/ports"
)

type mockPerceptionPort struct {
	normalizeResult domain.Payload
	normalizeErr    error
}

func (m *mockPerceptionPort) Normalize(ctx context.Context, signal domain.Signal) (domain.Payload, error) {
	return m.normalizeResult, m.normalizeErr
}

type mockGenerativePort struct {
	generateResult *ports.Plan
	generateErr    error
}

func (m *mockGenerativePort) Generate(ctx context.Context, intent domain.IntentStr, compiledPrompt string, context []domain.Atom, genes []domain.DomainGene) (*ports.Plan, error) {
	return m.generateResult, m.generateErr
}

func (m *mockGenerativePort) Induce(ctx context.Context, input string) (string, error) {
	return "", nil
}

func (m *mockGenerativePort) Embed(ctx context.Context, text string) (ports.Vector, error) {
	return nil, nil
}

type mockCompilerPort struct {
	compileResult string
	compileErr     error
}

func (m *mockCompilerPort) Compile(ctx context.Context, intent domain.IntentStr, options ...interface{}) (string, error) {
	return m.compileResult, m.compileErr
}

type mockAuditorPortForOrch struct {
	verifyResult *domain.AuditResult
	verifyErr    error
}

func (m *mockAuditorPortForOrch) Verify(ctx context.Context, frame *domain.CognitiveFrame) (*domain.AuditResult, error) {
	return m.verifyResult, m.verifyErr
}

func (m *mockAuditorPortForOrch) GenerateTrace(frame *domain.CognitiveFrame) []byte {
	return []byte("trace")
}

type mockGenePoolPortForOrch struct {
	genes []domain.DomainGene
}

func (m *mockGenePoolPortForOrch) ActiveGenes(ctx context.Context, intent domain.IntentStr) iter.Seq[*domain.DomainGene] {
	return func(yield func(*domain.DomainGene) bool) {
		for _, g := range m.genes {
			g := g
			yield(&g)
		}
	}
}

func (m *mockGenePoolPortForOrch) Reload(ctx context.Context) error {
	return nil
}

type mockGenomeStoragePort struct {
	persistTraceErr error
}

func (m *mockGenomeStoragePort) ReadManifest(ctx context.Context, path string) ([]byte, error) { return nil, nil }
func (m *mockGenomeStoragePort) MapGene(ctx context.Context, path string) ([]byte, uintptr, error) { return nil, 0, nil }
func (m *mockGenomeStoragePort) UnmapGene(data []byte) error { return nil }
func (m *mockGenomeStoragePort) CalculateFileHash(ctx context.Context, path string) (string, error) { return "", nil }
func (m *mockGenomeStoragePort) ReadFile(ctx context.Context, path string) ([]byte, error) { return nil, nil }
func (m *mockGenomeStoragePort) WriteFile(ctx context.Context, path string, data []byte, perm os.FileMode) error { return nil }
func (m *mockGenomeStoragePort) LoadKnowledge(ctx context.Context, intent string) ([]byte, error) { return nil, nil }
func (m *mockGenomeStoragePort) PersistKnowledge(ctx context.Context, intent string, data []byte) error { return nil }
func (m *mockGenomeStoragePort) PersistTrace(ctx context.Context, frame *domain.CognitiveFrame, content []byte) error {
	return m.persistTraceErr
}
func (m *mockGenomeStoragePort) PersistProposal(ctx context.Context, intent string, data []byte) error { return nil }
func (m *mockGenomeStoragePort) PersistAsync(ctx context.Context, path string, data []byte) error { return nil }
func (m *mockGenomeStoragePort) Flush(ctx context.Context) error { return nil }
func (m *mockGenomeStoragePort) ResolvePath(kind, id string) string { return "" }

func makePayload(atoms []domain.Atom) domain.Payload {
	return func(yield func(domain.Atom) bool) {
		for _, a := range atoms {
			if !yield(a) {
				return
			}
		}
	}
}

func TestOrchestrator_Execute_PassOnFirstAttempt(t *testing.T) {
	perception := &mockPerceptionPort{
		normalizeResult: makePayload([]domain.Atom{{Predicate: "obs", Subject: "s", Object: "o"}}),
	}
	proposer := &mockGenerativePort{
		generateResult: &ports.Plan{Intent: "test", Steps: []map[string]any{{"step": "1"}}},
	}
	compiler := &mockCompilerPort{
		compileResult: "compiled prompt",
	}
	auditor := &mockAuditorPortForOrch{
		verifyResult: &domain.AuditResult{Pass: true},
	}
	genePool := &mockGenePoolPortForOrch{
		genes: []domain.DomainGene{
			{Name: "kernel", Tier: domain.Tier0Kernel},
		},
	}
	storage := &mockGenomeStoragePort{}

	orch := New(perception, proposer, compiler, auditor, genePool, storage)

	signal := domain.Signal{Intent: domain.IntentStr("test-signal")}
	output, err := orch.Execute(context.Background(), signal)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output.Action != "EXECUTE_PLAN" {
		t.Errorf("expected action EXECUTE_PLAN, got %s", output.Action)
	}
}

func TestOrchestrator_Execute_ObserveFails(t *testing.T) {
	perception := &mockPerceptionPort{
		normalizeErr: errors.New("observe failed"),
	}
	proposer := &mockGenerativePort{}
	compiler := &mockCompilerPort{}
	auditor := &mockAuditorPortForOrch{}
	genePool := &mockGenePoolPortForOrch{}
	storage := &mockGenomeStoragePort{}

	orch := New(perception, proposer, compiler, auditor, genePool, storage)

	signal := domain.Signal{Intent: domain.IntentStr("test")}
	_, err := orch.Execute(context.Background(), signal)
	if err == nil {
		t.Fatal("expected error from observe phase")
	}
}

func TestOrchestrator_Execute_CompileFails(t *testing.T) {
	perception := &mockPerceptionPort{
		normalizeResult: makePayload([]domain.Atom{{Predicate: "obs"}}),
	}
	proposer := &mockGenerativePort{}
	compiler := &mockCompilerPort{
		compileErr: errors.New("compile failed"),
	}
	auditor := &mockAuditorPortForOrch{}
	genePool := &mockGenePoolPortForOrch{
		genes: []domain.DomainGene{{Name: "kernel", Tier: domain.Tier0Kernel}},
	}
	storage := &mockGenomeStoragePort{}

	orch := New(perception, proposer, compiler, auditor, genePool, storage)

	signal := domain.Signal{Intent: domain.IntentStr("test")}
	_, err := orch.Execute(context.Background(), signal)
	if err == nil {
		t.Fatal("expected error from compile phase")
	}
}

func TestOrchestrator_Execute_GenerateFails(t *testing.T) {
	perception := &mockPerceptionPort{
		normalizeResult: makePayload([]domain.Atom{{Predicate: "obs"}}),
	}
	proposer := &mockGenerativePort{
		generateErr: errors.New("generate failed"),
	}
	compiler := &mockCompilerPort{
		compileResult: "prompt",
	}
	auditor := &mockAuditorPortForOrch{}
	genePool := &mockGenePoolPortForOrch{
		genes: []domain.DomainGene{{Name: "kernel", Tier: domain.Tier0Kernel}},
	}
	storage := &mockGenomeStoragePort{}

	orch := New(perception, proposer, compiler, auditor, genePool, storage)

	signal := domain.Signal{Intent: domain.IntentStr("test")}
	_, err := orch.Execute(context.Background(), signal)
	if err == nil {
		t.Fatal("expected error from generate phase")
	}
}

func TestOrchestrator_Execute_DeadlockAfterMaxRefinements(t *testing.T) {
	perception := &mockPerceptionPort{
		normalizeResult: makePayload([]domain.Atom{{Predicate: "obs"}}),
	}
	proposer := &mockGenerativePort{
		generateResult: &ports.Plan{Intent: "test", Steps: []map[string]any{{"step": "1"}}},
	}
	compiler := &mockCompilerPort{
		compileResult: "prompt",
	}
	auditor := &mockAuditorPortForOrch{
		verifyResult: &domain.AuditResult{Pass: false, ViolationTier: domain.Tier0Kernel},
	}
	genePool := &mockGenePoolPortForOrch{
		genes: []domain.DomainGene{{Name: "kernel", Tier: domain.Tier0Kernel}},
	}
	storage := &mockGenomeStoragePort{}

	orch := New(perception, proposer, compiler, auditor, genePool, storage)

	signal := domain.Signal{Intent: domain.IntentStr("test")}
	_, err := orch.Execute(context.Background(), signal)
	if err == nil {
		t.Fatal("expected deadlock error after max refinements")
	}
}

func TestOrchestrator_Execute_AuditorSystemError(t *testing.T) {
	perception := &mockPerceptionPort{
		normalizeResult: makePayload([]domain.Atom{{Predicate: "obs"}}),
	}
	proposer := &mockGenerativePort{
		generateResult: &ports.Plan{Intent: "test", Steps: []map[string]any{{"step": "1"}}},
	}
	compiler := &mockCompilerPort{
		compileResult: "prompt",
	}
	auditor := &mockAuditorPortForOrch{
		verifyErr: errors.New("auditor system error"),
	}
	genePool := &mockGenePoolPortForOrch{
		genes: []domain.DomainGene{{Name: "kernel", Tier: domain.Tier0Kernel}},
	}
	storage := &mockGenomeStoragePort{}

	orch := New(perception, proposer, compiler, auditor, genePool, storage)

	signal := domain.Signal{Intent: domain.IntentStr("test")}
	_, err := orch.Execute(context.Background(), signal)
	if err == nil {
		t.Fatal("expected error from auditor")
	}
}

func TestOrchestrator_Execute_RefinementPasses(t *testing.T) {
	passCount := 0
	perception := &mockPerceptionPort{
		normalizeResult: makePayload([]domain.Atom{{Predicate: "obs"}}),
	}
	proposer := &mockGenerativePort{
		generateResult: &ports.Plan{Intent: "test", Steps: []map[string]any{{"step": "1"}}},
	}
	compiler := &mockCompilerPort{
		compileResult: "prompt",
	}
	auditor := &mockAuditorPortForOrch{
		verifyResult: &domain.AuditResult{Pass: true},
	}
	genePool := &mockGenePoolPortForOrch{
		genes: []domain.DomainGene{{Name: "kernel", Tier: domain.Tier0Kernel}},
	}
	storage := &mockGenomeStoragePort{}

	orch := New(perception, proposer, compiler, auditor, genePool, storage)
	_ = passCount

	signal := domain.Signal{Intent: domain.IntentStr("test")}
	output, err := orch.Execute(context.Background(), signal)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output.Action != "EXECUTE_PLAN" {
		t.Errorf("expected action EXECUTE_PLAN, got %s", output.Action)
	}
	_ = passCount
}

func TestOrchestrator_FinalizeOutput_NilPlan(t *testing.T) {
	orch := &Orchestrator{}
	output := orch.finalizeOutput(nil)
	if output.Action != "NO_OP" {
		t.Errorf("expected NO_OP for nil plan, got %s", output.Action)
	}
}

func TestOrchestrator_FinalizeOutput_EmptySteps(t *testing.T) {
	orch := &Orchestrator{}
	output := orch.finalizeOutput(&ports.Plan{Steps: []map[string]any{}})
	if output.Action != "NO_OP" {
		t.Errorf("expected NO_OP for empty steps, got %s", output.Action)
	}
}

func TestOrchestrator_FinalizeOutput_ValidPlan(t *testing.T) {
	orch := &Orchestrator{}
	output := orch.finalizeOutput(&ports.Plan{
		Intent: "test",
		Steps:  []map[string]any{{"action": "do something"}},
	})
	if output.Action != "EXECUTE_PLAN" {
		t.Errorf("expected EXECUTE_PLAN, got %s", output.Action)
	}
	if output.Params == nil {
		t.Error("expected params to be set")
	}
}

func TestOrchestrator_Execute_NoGenes(t *testing.T) {
	perception := &mockPerceptionPort{
		normalizeResult: makePayload([]domain.Atom{}),
	}
	proposer := &mockGenerativePort{
		generateResult: &ports.Plan{Intent: "test", Steps: []map[string]any{{"step": "1"}}},
	}
	compiler := &mockCompilerPort{
		compileResult: "prompt",
	}
	auditor := &mockAuditorPortForOrch{
		verifyResult: &domain.AuditResult{Pass: true},
	}
	genePool := &mockGenePoolPortForOrch{
		genes: []domain.DomainGene{},
	}
	storage := &mockGenomeStoragePort{}

	orch := New(perception, proposer, compiler, auditor, genePool, storage)

	signal := domain.Signal{Intent: domain.IntentStr("test")}
	_, err := orch.Execute(context.Background(), signal)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}