package genepool

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/duynguyendang/manglekit/internal/core/domain"
)

// mockStorage implements ports.GenomeStoragePort for testing.
type mockStorage struct {
	manifestData []byte
	manifestErr  error
	geneData     []byte
	geneAddr     uintptr
	geneErr      error
	hashValue    string
	hashErr      error
}

func (m *mockStorage) ReadManifest(ctx context.Context, path string) ([]byte, error) {
	return m.manifestData, m.manifestErr
}

func (m *mockStorage) MapGene(ctx context.Context, path string) ([]byte, uintptr, error) {
	return m.geneData, m.geneAddr, m.geneErr
}

func (m *mockStorage) UnmapGene(data []byte) error { return nil }

func (m *mockStorage) CalculateFileHash(ctx context.Context, path string) (string, error) {
	return m.hashValue, m.hashErr
}

func (m *mockStorage) ReadFile(ctx context.Context, path string) ([]byte, error) {
	return nil, nil
}
func (m *mockStorage) WriteFile(ctx context.Context, path string, data []byte, perm os.FileMode) error {
	return nil
}
func (m *mockStorage) LoadKnowledge(ctx context.Context, intent string) ([]byte, error) {
	return nil, nil
}
func (m *mockStorage) PersistKnowledge(ctx context.Context, intent string, data []byte) error {
	return nil
}
func (m *mockStorage) PersistTrace(ctx context.Context, frame *domain.CognitiveFrame, content []byte) error {
	return nil
}
func (m *mockStorage) PersistProposal(ctx context.Context, intent string, data []byte) error {
	return nil
}
func (m *mockStorage) PersistAsync(ctx context.Context, path string, data []byte) error {
	return nil
}
func (m *mockStorage) Flush(ctx context.Context) error { return nil }
func (m *mockStorage) ResolvePath(kind, id string) string {
	return fmt.Sprintf("/tmp/%s/%s", kind, id)
}

var validManifest = []byte(`genes:
  - name: test_gene
    path: /tmp/test.dlog
    tier: "T1"
    signature: "abc123"
    capabilities:
      - audit
    intents:
      - test_intent
`)

var multiGeneManifest = []byte(`genes:
  - name: gene_a
    path: /tmp/a.dlog
    tier: "T0"
    capabilities:
      - guard
    intents:
      - intent_a
  - name: gene_b
    path: /tmp/b.dlog
    tier: "T2"
    capabilities:
      - enrich
    intents:
      - intent_b
  - name: gene_c
    path: /tmp/c.dlog
    tier: "T1"
    capabilities:
      - audit
`)

func TestGenePool_New_Success(t *testing.T) {
	storage := &mockStorage{
		manifestData: validManifest,
		geneData:     []byte("rule content"),
		geneAddr:     0x1000,
		hashValue:    "abc123",
	}

	gp, err := New(context.Background(), storage, "/tmp/manifest.yaml")
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	if gp == nil {
		t.Fatal("expected non-nil GenePool")
	}
}

func TestGenePool_New_EmptyManifest(t *testing.T) {
	storage := &mockStorage{
		manifestData: []byte{},
	}

	gp, err := New(context.Background(), storage, "/tmp/manifest.yaml")
	if err != nil {
		t.Fatalf("New() with empty manifest should succeed, got: %v", err)
	}
	if gp == nil {
		t.Fatal("expected non-nil GenePool")
	}
}

func TestGenePool_New_ManifestReadError(t *testing.T) {
	storage := &mockStorage{
		manifestErr: fmt.Errorf("permission denied"),
	}

	gp, err := New(context.Background(), storage, "/tmp/manifest.yaml")
	// When manifest read fails, Reload returns nil (empty pool)
	// because the auditor will catch Tier 0 absence later.
	if err != nil {
		t.Fatalf("New() with manifest read error should succeed (empty pool), got: %v", err)
	}
	if gp == nil {
		t.Fatal("expected non-nil GenePool")
	}
}

func TestGenePool_New_InvalidYAML(t *testing.T) {
	storage := &mockStorage{
		manifestData: []byte("not valid yaml: ["),
	}

	_, err := New(context.Background(), storage, "/tmp/manifest.yaml")
	if err == nil {
		t.Fatal("expected error on invalid YAML")
	}
}

func TestGenePool_New_SignatureMismatch(t *testing.T) {
	storage := &mockStorage{
		manifestData: validManifest,
		geneData:     []byte("rule content"),
		hashValue:    "wrong_hash",
	}

	_, err := New(context.Background(), storage, "/tmp/manifest.yaml")
	if err == nil {
		t.Fatal("expected error on signature mismatch")
	}
}

func TestGenePool_New_MapGeneError(t *testing.T) {
	storage := &mockStorage{
		manifestData: validManifest,
		geneErr:      fmt.Errorf("file not found"),
	}

	_, err := New(context.Background(), storage, "/tmp/manifest.yaml")
	if err == nil {
		t.Fatal("expected error on MapGene failure")
	}
}

func TestGenePool_ActiveGenes_FilterByIntent(t *testing.T) {
	storage := &mockStorage{
		manifestData: multiGeneManifest,
		geneData:     []byte("rules"),
		geneAddr:     0x1000,
		hashValue:    "",
	}

	gp, err := New(context.Background(), storage, "/tmp/manifest.yaml")
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	// Query with intent_a — should get gene_a (intent match) and gene_c (no intent filter)
	var genes []string
	for gene := range gp.ActiveGenes(context.Background(), "intent_a") {
		genes = append(genes, gene.Name)
	}

	if len(genes) != 2 {
		t.Fatalf("expected 2 genes for intent_a (gene_a + gene_c), got %d: %v", len(genes), genes)
	}

	// Verify gene_a is present (has matching intent)
	foundA := false
	for _, name := range genes {
		if name == "gene_a" {
			foundA = true
		}
	}
	if !foundA {
		t.Error("gene_a should be present for intent_a")
	}

	// Verify gene_c is present (no intent filter)
	foundC := false
	for _, name := range genes {
		if name == "gene_c" {
			foundC = true
		}
	}
	if !foundC {
		t.Error("gene_c (no intent filter) should be present for any intent")
	}
}

func TestGenePool_ActiveGenes_NoFilterOnEmptyIntent(t *testing.T) {
	storage := &mockStorage{
		manifestData: multiGeneManifest,
		geneData:     []byte("rules"),
		geneAddr:     0x1000,
		hashValue:    "",
	}

	gp, err := New(context.Background(), storage, "/tmp/manifest.yaml")
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	// gene_c has no intents, so it should be returned for any intent
	var genes []string
	for gene := range gp.ActiveGenes(context.Background(), "any_intent") {
		genes = append(genes, gene.Name)
	}

	// gene_c has no intent filter, so it matches all intents
	found := false
	for _, name := range genes {
		if name == "gene_c" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("gene_c (no intent filter) should match any intent, got: %v", genes)
	}
}

func TestGenePool_ActiveGenes_Empty(t *testing.T) {
	storage := &mockStorage{
		manifestData: []byte{},
	}

	gp, err := New(context.Background(), storage, "/tmp/manifest.yaml")
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	count := 0
	for range gp.ActiveGenes(context.Background(), "any") {
		count++
	}
	if count != 0 {
		t.Errorf("expected 0 genes from empty pool, got %d", count)
	}
}

func TestGenePool_Reload_UpdatesGenes(t *testing.T) {
	storage := &mockStorage{
		manifestData: validManifest,
		geneData:     []byte("rules v1"),
		geneAddr:     0x1000,
		hashValue:    "abc123",
	}

	gp, err := New(context.Background(), storage, "/tmp/manifest.yaml")
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	// Verify initial state
	count := 0
	for range gp.ActiveGenes(context.Background(), "test_intent") {
		count++
	}
	if count != 1 {
		t.Fatalf("expected 1 gene initially, got %d", count)
	}

	// Now reload with a different manifest
	storage.manifestData = multiGeneManifest
	storage.hashValue = ""

	if err := gp.Reload(context.Background()); err != nil {
		t.Fatalf("Reload() failed: %v", err)
	}

	// Verify updated state — gene_a should now be available for intent_a
	count = 0
	for range gp.ActiveGenes(context.Background(), "intent_a") {
		count++
	}
	// gene_a matches intent_a, gene_c matches any intent (no filter)
	if count != 2 {
		t.Errorf("expected 2 genes after reload (gene_a + gene_c), got %d", count)
	}
}

func TestGenePool_ReloadTrigger_DelegatesToReload(t *testing.T) {
	storage := &mockStorage{
		manifestData: validManifest,
		geneData:     []byte("rules"),
		geneAddr:     0x1000,
		hashValue:    "abc123",
	}

	gp, err := New(context.Background(), storage, "/tmp/manifest.yaml")
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	// ReloadTrigger should succeed (manifest is still valid)
	if err := gp.ReloadTrigger(); err != nil {
		t.Errorf("ReloadTrigger() failed: %v", err)
	}
}
