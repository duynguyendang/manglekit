package proposer

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/duynguyendang/manglekit/internal/core/domain"
	"github.com/duynguyendang/manglekit/internal/core/ports"
)

type mockGenerativePortForProp struct {
	generateResult *ports.Plan
	generateErr    error
	embedResult    ports.Vector
	embedErr       error
	induceResult   string
	induceErr      error
}

func (m *mockGenerativePortForProp) Generate(ctx context.Context, intent domain.IntentStr, compiledPrompt string, baseContext []domain.Atom, genes []domain.DomainGene) (*ports.Plan, error) {
	return m.generateResult, m.generateErr
}

func (m *mockGenerativePortForProp) Embed(ctx context.Context, text string) (ports.Vector, error) {
	return m.embedResult, m.embedErr
}

func (m *mockGenerativePortForProp) Induce(ctx context.Context, input string) (string, error) {
	return m.induceResult, m.induceErr
}

type mockVectorStorePort struct {
	searchResult []string
	searchErr     error
}

func (m *mockVectorStorePort) Insert(vector ports.Vector, metadata string) error {
	return nil
}

func (m *mockVectorStorePort) Search(vector ports.Vector, limit int, threshold float64) ([]string, error) {
	return m.searchResult, m.searchErr
}

type mockGenomeStoragePortForProp struct {
	readFileResult []byte
	readFileErr    error
}

func (m *mockGenomeStoragePortForProp) ResolvePath(kind, id string) string {
	if kind == "knowledge" && id != "" {
		return "/knowledge/" + id
	}
	return ""
}

func (m *mockGenomeStoragePortForProp) ReadFile(ctx context.Context, path string) ([]byte, error) {
	return m.readFileResult, m.readFileErr
}

func (m *mockGenomeStoragePortForProp) ReadManifest(ctx context.Context, path string) ([]byte, error) { return nil, nil }
func (m *mockGenomeStoragePortForProp) MapGene(ctx context.Context, path string) ([]byte, uintptr, error) { return nil, 0, nil }
func (m *mockGenomeStoragePortForProp) UnmapGene(data []byte) error { return nil }
func (m *mockGenomeStoragePortForProp) CalculateFileHash(ctx context.Context, path string) (string, error) { return "", nil }
func (m *mockGenomeStoragePortForProp) WriteFile(ctx context.Context, path string, data []byte, perm os.FileMode) error { return nil }
func (m *mockGenomeStoragePortForProp) LoadKnowledge(ctx context.Context, intent string) ([]byte, error) { return nil, nil }
func (m *mockGenomeStoragePortForProp) PersistKnowledge(ctx context.Context, intent string, data []byte) error { return nil }
func (m *mockGenomeStoragePortForProp) PersistTrace(ctx context.Context, frame *domain.CognitiveFrame, content []byte) error { return nil }
func (m *mockGenomeStoragePortForProp) PersistProposal(ctx context.Context, intent string, data []byte) error { return nil }
func (m *mockGenomeStoragePortForProp) PersistAsync(ctx context.Context, path string, data []byte) error { return nil }
func (m *mockGenomeStoragePortForProp) Flush(ctx context.Context) error { return nil }

type mockEmbeddingPort struct {
	embedResult ports.Vector
	embedErr    error
}

func (m *mockEmbeddingPort) Embed(ctx context.Context, text string) (ports.Vector, error) {
	return m.embedResult, m.embedErr
}

func TestService_Generate_WithVectorMatches(t *testing.T) {
	llm := &mockGenerativePortForProp{
		generateResult: &ports.Plan{Intent: "test", Steps: []map[string]any{{"step": "1"}}},
	}
	vector := &mockVectorStorePort{
		searchResult: []string{"match1", "match2"},
	}
	storage := &mockGenomeStoragePortForProp{}
	embedder := &mockEmbeddingPort{
		embedResult: ports.Vector{0.1, 0.2, 0.3},
	}

	svc := NewService(llm, vector, storage, embedder)

	atoms, err := svc.Generate(context.Background(), "test-intent", "prompt", []domain.Atom{}, []domain.DomainGene{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if atoms == nil {
		t.Fatal("expected non-nil plan")
	}
	if len(atoms.Steps) != 1 {
		t.Errorf("expected 1 step, got %d", len(atoms.Steps))
	}
}

func TestService_Generate_NoVectorResults(t *testing.T) {
	llm := &mockGenerativePortForProp{
		generateResult: &ports.Plan{Intent: "test", Steps: []map[string]any{{"step": "1"}}},
	}
	vector := &mockVectorStorePort{
		searchResult: nil,
	}
	storage := &mockGenomeStoragePortForProp{}
	embedder := &mockEmbeddingPort{
		embedResult: ports.Vector{0.1, 0.2},
	}

	svc := NewService(llm, vector, storage, embedder)

	_, err := svc.Generate(context.Background(), "test-intent", "prompt", []domain.Atom{}, []domain.DomainGene{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestService_Generate_EmbedErrorContinues(t *testing.T) {
	llm := &mockGenerativePortForProp{
		generateResult: &ports.Plan{Intent: "test", Steps: []map[string]any{{"step": "1"}}},
	}
	vector := &mockVectorStorePort{}
	storage := &mockGenomeStoragePortForProp{}
	embedder := &mockEmbeddingPort{
		embedErr: errors.New("embed failed"),
	}

	svc := NewService(llm, vector, storage, embedder)

	_, err := svc.Generate(context.Background(), "test-intent", "prompt", []domain.Atom{}, []domain.DomainGene{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestService_Generate_WithFileFacts(t *testing.T) {
	llm := &mockGenerativePortForProp{
		generateResult: &ports.Plan{Intent: "test", Steps: []map[string]any{{"step": "1"}}},
	}
	vector := &mockVectorStorePort{}
	storage := &mockGenomeStoragePortForProp{
		readFileResult: []byte("fact content"),
	}
	embedder := &mockEmbeddingPort{
		embedErr: errors.New("embed failed"),
	}

	svc := NewService(llm, vector, storage, embedder)

	_, err := svc.Generate(context.Background(), "test-intent", "prompt", []domain.Atom{}, []domain.DomainGene{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestService_Generate_NoFileFacts(t *testing.T) {
	llm := &mockGenerativePortForProp{
		generateResult: &ports.Plan{Intent: "test", Steps: []map[string]any{{"step": "1"}}},
	}
	vector := &mockVectorStorePort{}
	storage := &mockGenomeStoragePortForProp{
		readFileErr: errors.New("file not found"),
	}
	embedder := &mockEmbeddingPort{
		embedErr: errors.New("embed failed"),
	}

	svc := NewService(llm, vector, storage, embedder)

	_, err := svc.Generate(context.Background(), "test-intent", "prompt", []domain.Atom{}, []domain.DomainGene{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestService_Generate_LLMGenerateError(t *testing.T) {
	llm := &mockGenerativePortForProp{
		generateErr: errors.New("generate failed"),
	}
	vector := &mockVectorStorePort{}
	storage := &mockGenomeStoragePortForProp{}
	embedder := &mockEmbeddingPort{
		embedResult: ports.Vector{0.1},
	}

	svc := NewService(llm, vector, storage, embedder)

	_, err := svc.Generate(context.Background(), "test-intent", "prompt", []domain.Atom{}, []domain.DomainGene{})
	if err == nil {
		t.Fatal("expected error from LLM")
	}
}

func TestService_Embed_Delegates(t *testing.T) {
	llm := &mockGenerativePortForProp{
		embedResult: ports.Vector{0.1, 0.2},
	}
	vector := &mockVectorStorePort{}
	storage := &mockGenomeStoragePortForProp{}
	embedder := &mockEmbeddingPort{}

	svc := NewService(llm, vector, storage, embedder)

	result, err := svc.Embed(context.Background(), "test text")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 elements, got %d", len(result))
	}
}

func TestService_Induce_Delegates(t *testing.T) {
	llm := &mockGenerativePortForProp{
		induceResult: "induced rules",
	}
	vector := &mockVectorStorePort{}
	storage := &mockGenomeStoragePortForProp{}
	embedder := &mockEmbeddingPort{}

	svc := NewService(llm, vector, storage, embedder)

	result, err := svc.Induce(context.Background(), "raw content")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "induced rules" {
		t.Errorf("expected 'induced rules', got %q", result)
	}
}