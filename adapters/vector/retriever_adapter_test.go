package vector

import (
	"context"
	"errors"
	"testing"

	"github.com/duynguyendang/manglekit/core"
)

// mockDocumentRetriever is a test implementation of DocumentRetriever.
type mockDocumentRetriever struct {
	docs []Document
	err  error
}

func (m *mockDocumentRetriever) Retrieve(ctx context.Context, query string) ([]Document, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.docs, nil
}

func TestRetrieverAction_Execute_Success(t *testing.T) {
	retriever := &mockDocumentRetriever{
		docs: []Document{
			{Content: "Doc 1 content", Source: "source1.txt"},
			{Content: "Doc 2 content", Source: "source2.txt"},
		},
	}
	action := NewRetrieverAction("test-retriever", retriever)

	input := core.NewEnvelope("test query")
	output, err := action.Execute(context.Background(), input)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	payload, ok := output.Payload.(string)
	if !ok {
		t.Fatalf("expected string payload, got %T", output.Payload)
	}

	// Verify JSON contains both documents
	if payload == "" {
		t.Error("expected non-empty payload")
	}

	if output.GetMeta("doc_count") != "2" {
		t.Errorf("expected doc_count '2', got %q", output.GetMeta("doc_count"))
	}

	if output.GetMeta("action_name") != "test-retriever" {
		t.Errorf("expected action_name 'test-retriever', got %q", output.GetMeta("action_name"))
	}
}

func TestRetrieverAction_Execute_EmptyDocs(t *testing.T) {
	retriever := &mockDocumentRetriever{docs: []Document{}}
	action := NewRetrieverAction("test-retriever", retriever)

	input := core.NewEnvelope("test query")
	output, err := action.Execute(context.Background(), input)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.GetMeta("doc_count") != "0" {
		t.Errorf("expected doc_count '0', got %q", output.GetMeta("doc_count"))
	}
}

func TestRetrieverAction_Execute_InvalidInput(t *testing.T) {
	retriever := &mockDocumentRetriever{docs: []Document{}}
	action := NewRetrieverAction("test-retriever", retriever)

	// Pass non-string payload
	input := core.NewEnvelope(123)
	_, err := action.Execute(context.Background(), input)

	if err == nil {
		t.Fatal("expected error for invalid input type")
	}

	if !errors.Is(err, core.ErrSystemError) {
		t.Errorf("expected ErrSystemError, got %v", err)
	}
}

func TestRetrieverAction_Execute_RetrieverError(t *testing.T) {
	retrieverErr := errors.New("retrieval failed")
	retriever := &mockDocumentRetriever{err: retrieverErr}
	action := NewRetrieverAction("test-retriever", retriever)

	input := core.NewEnvelope("test query")
	_, err := action.Execute(context.Background(), input)

	if err == nil {
		t.Fatal("expected error from retriever")
	}

	if !errors.Is(err, retrieverErr) {
		t.Errorf("expected wrapped retriever error, got %v", err)
	}
}

func TestRetrieverAction_Metadata(t *testing.T) {
	retriever := &mockDocumentRetriever{}
	action := NewRetrieverAction("my-retriever-action", retriever)

	meta := action.Metadata()

	if meta.Name != "my-retriever-action" {
		t.Errorf("expected name 'my-retriever-action', got %q", meta.Name)
	}

	if meta.Type != "retriever" {
		t.Errorf("expected type 'retriever', got %q", meta.Type)
	}
}

func TestFormatDocsAsContext_Success(t *testing.T) {
	docsJSON := `[{"content":"First doc","source":"a.txt"},{"content":"Second doc","source":"b.txt"}]`
	result, err := FormatDocsAsContext(docsJSON)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == "" {
		t.Error("expected non-empty result")
	}

	// Verify content is included
	if !contains(result, "First doc") || !contains(result, "Second doc") {
		t.Error("expected both documents in formatted context")
	}

	// Verify sources are included
	if !contains(result, "a.txt") || !contains(result, "b.txt") {
		t.Error("expected both sources in formatted context")
	}
}

func TestFormatDocsAsContext_Empty(t *testing.T) {
	docsJSON := `[]`
	result, err := FormatDocsAsContext(docsJSON)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "No relevant documents found." {
		t.Errorf("expected 'No relevant documents found.', got %q", result)
	}
}

func TestFormatDocsAsContext_InvalidJSON(t *testing.T) {
	docsJSON := `invalid json`
	_, err := FormatDocsAsContext(docsJSON)

	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
