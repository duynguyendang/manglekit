package vectorstores

import (
	"context"
	"errors"
	"testing"

	"github.com/duynguyendang/manglekit/core"
)

// MockRetriever is a mock implementation of core.Retriever for testing.
type MockRetriever struct {
	docs []core.Doc
	err  error
}

func (m *MockRetriever) Retrieve(ctx context.Context, req core.RetrieveRequest) (core.RetrieveResult, error) {
	if m.err != nil {
		return core.RetrieveResult{}, m.err
	}
	return core.RetrieveResult{
		Docs: m.docs,
		Meta: map[string]any{},
	}, nil
}

// MockLogger is a mock implementation of core.Logger for testing.
type MockLogger struct {
	debugMessages []string
	infoMessages  []string
	warnMessages  []string
	errorMessages []string
}

func (m *MockLogger) Debugf(msg string, kv ...any) {
	m.debugMessages = append(m.debugMessages, msg)
}

func (m *MockLogger) Infof(msg string, kv ...any) {
	m.infoMessages = append(m.infoMessages, msg)
}

func (m *MockLogger) Warnf(msg string, kv ...any) {
	m.warnMessages = append(m.warnMessages, msg)
}

func (m *MockLogger) Errorf(msg string, kv ...any) {
	m.errorMessages = append(m.errorMessages, msg)
}

func (m *MockLogger) With(kv ...any) core.Logger {
	return m
}

// TestGenkitRetrieverAdapterSearch tests the Search method of genkitRetrieverAdapter.
func TestGenkitRetrieverAdapterSearch(t *testing.T) {
	testDocs := []core.Doc{
		{
			ID:     "doc1",
			Text:   "First document",
			Source: "file1.txt",
			URI:    "file:///path/to/file1.txt",
			Meta:   map[string]any{"author": "Alice"},
		},
		{
			ID:     "doc2",
			Text:   "Second document",
			Source: "file2.txt",
			URI:    "file:///path/to/file2.txt",
			Meta:   map[string]any{"author": "Bob"},
		},
	}

	mockRetriever := &MockRetriever{docs: testDocs}
	mockLogger := &MockLogger{}
	adapter := newGenkitRetrieverAdapter(mockRetriever, mockLogger)

	ctx := context.Background()
	results, err := adapter.Search(ctx, "test query", nil, 10, nil)

	if err != nil {
		t.Fatalf("Search failed unexpectedly: %v", err)
	}

	if len(results) != len(testDocs) {
		t.Errorf("expected %d documents, got %d", len(testDocs), len(results))
	}

	for i, doc := range results {
		if doc.ID != testDocs[i].ID {
			t.Errorf("doc %d: expected ID %q, got %q", i, testDocs[i].ID, doc.ID)
		}
		if doc.Text != testDocs[i].Text {
			t.Errorf("doc %d: expected text %q, got %q", i, testDocs[i].Text, doc.Text)
		}
	}
}

// TestGenkitRetrieverAdapterSearchError tests that Search properly propagates errors.
func TestGenkitRetrieverAdapterSearchError(t *testing.T) {
	testError := errors.New("retriever error")
	mockRetriever := &MockRetriever{err: testError}
	mockLogger := &MockLogger{}
	adapter := newGenkitRetrieverAdapter(mockRetriever, mockLogger)

	ctx := context.Background()
	_, err := adapter.Search(ctx, "test query", nil, 10, nil)

	if err == nil {
		t.Fatal("expected Search to return an error")
	}

	if !errors.Is(err, testError) {
		t.Errorf("expected error to contain %v, got %v", testError, err)
	}
}

// TestGenkitRetrieverAdapterAddDocumentsNotSupported tests that AddDocuments returns ErrNotSupported.
func TestGenkitRetrieverAdapterAddDocumentsNotSupported(t *testing.T) {
	mockRetriever := &MockRetriever{}
	mockLogger := &MockLogger{}
	adapter := newGenkitRetrieverAdapter(mockRetriever, mockLogger)

	ctx := context.Background()
	docs := []core.Doc{
		{ID: "doc1", Text: "text1"},
	}

	err := adapter.AddDocuments(ctx, docs)

	if err != core.ErrNotSupported {
		t.Errorf("expected ErrNotSupported, got %v", err)
	}

	// Verify that a debug message was logged
	if len(mockLogger.debugMessages) == 0 {
		t.Errorf("expected debug message to be logged")
	}
	if mockLogger.debugMessages[0] != "AddDocuments called on read-only vector store (Genkit-delegated retriever)" {
		t.Errorf("unexpected debug message: %s", mockLogger.debugMessages[0])
	}
}

// TestGenkitRetrieverAdapterAddDocumentsNoLogger tests AddDocuments when logger is nil.
func TestGenkitRetrieverAdapterAddDocumentsNoLogger(t *testing.T) {
	mockRetriever := &MockRetriever{}
	adapter := newGenkitRetrieverAdapter(mockRetriever, nil)

	ctx := context.Background()
	docs := []core.Doc{
		{ID: "doc1", Text: "text1"},
	}

	err := adapter.AddDocuments(ctx, docs)

	if err != core.ErrNotSupported {
		t.Errorf("expected ErrNotSupported, got %v", err)
	}
}

// TestExtractProviderName tests the extractProviderName function.
func TestExtractProviderName(t *testing.T) {
	// Test with ProviderNameGetter implementation
	opts := &MockProviderOptions{name: "pinecone"}
	name := extractProviderName(opts)

	if name != "pinecone" {
		t.Errorf("expected provider name 'pinecone', got %q", name)
	}

	// Test with nil (should return empty string)
	name = extractProviderName(nil)
	if name != "" {
		t.Errorf("expected empty string for nil options, got %q", name)
	}
}

// MockProviderOptions implements ProviderNameGetter for testing.
type MockProviderOptions struct {
	name string
}

func (m *MockProviderOptions) ProviderName() string {
	return m.name
}

func (m *MockProviderOptions) ProviderKind() core.Kind {
	return core.KindVectorStore
}
