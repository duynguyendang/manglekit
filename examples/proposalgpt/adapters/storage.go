package adapters

import (
	"context"
	"fmt"
	"os"

	"github.com/duynguyendang/manglekit-wip/internal/core/domain"
	"github.com/duynguyendang/manglekit-wip/internal/core/ports"
)

// MockStorage implements GenomeStoragePort to skip actual mmap file reading for the initial demo.
type MockStorage struct {
	baseDir string
}

func NewMockStorage(dir string) *MockStorage {
	return &MockStorage{baseDir: dir}
}

func (m *MockStorage) MapGene(ctx context.Context, relativePath string) ([]byte, uintptr, error) {
	// Dummy Datalog program that tests the Tier 0 Auditor rule.
	dl := `
// System tier dummy rule
halt("Unauthorized System Mutation") :- 
    input_param("target_namespace", "system").
`
	return []byte(dl), 0, nil
}

func (m *MockStorage) UnmapGene(data []byte) error { return nil }

func (m *MockStorage) CalculateFileHash(ctx context.Context, relativePath string) (string, error) {
	return "dummy-hash", nil
}

func (m *MockStorage) ResolvePath(category, filename string) string {
	return fmt.Sprintf("%s/%s/%s", m.baseDir, category, filename)
}

func (m *MockStorage) WriteFile(ctx context.Context, absolutePath string, data []byte, perm os.FileMode) error {
	return nil
}
func (m *MockStorage) ReadFile(ctx context.Context, absolutePath string) ([]byte, error) {
	return nil, nil
}

func (m *MockStorage) ReadManifest(ctx context.Context, path string) ([]byte, error) { return nil, nil }
func (m *MockStorage) LoadKnowledge(ctx context.Context, intent string) ([]byte, error) {
	return nil, nil
}
func (m *MockStorage) PersistProposal(ctx context.Context, intent string, data []byte) error {
	return nil
}
func (m *MockStorage) PersistAsync(ctx context.Context, path string, data []byte) error { return nil }
func (m *MockStorage) Flush(ctx context.Context) error                                  { return nil }

func (m *MockStorage) PersistKnowledge(ctx context.Context, intent string, compiledRules []byte) error {
	fmt.Printf("[STORAGE] Mock Persisted %d bytes of induced rules for intent: %s\n", len(compiledRules), intent)
	return nil
}

func (m *MockStorage) PersistTrace(ctx context.Context, frame *domain.CognitiveFrame, trace []byte) error {
	fmt.Printf("[STORAGE] Mock Persisted Execution Trace for Epoch: %s\n", frame.ID)
	return nil
}

// MockEvidence implements EvidenceStorePort
type MockEvidence struct{}

func NewMockEvidence() *MockEvidence { return &MockEvidence{} }

func (m *MockEvidence) Save(ctx context.Context, item ports.EvidenceItem) error { return nil }
func (m *MockEvidence) FindSimilar(ctx context.Context, intent, content string, threshold float64) (string, bool) {
	return "", false
}
func (m *MockEvidence) GetContext(ctx context.Context, intent string, limit int) ([]ports.EvidenceItem, error) {
	return nil, nil
}
