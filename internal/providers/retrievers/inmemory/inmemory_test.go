package inmemory

import (
	"context"
	"testing"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/retrieve"
)

func TestInMemory(t *testing.T) {
	docs := []core.Doc{
		{ID: "1", Text: "doc one"},
		{ID: "2", Text: "doc two"},
	}
	r, err := New(retrieve.InMemoryOptions{Documents: docs})
	if err != nil {
		t.Fatalf("expected New to succeed, got %v", err)
	}

	res, err := r.Retrieve(context.Background(), retrieve.Request{Query: "one"})
	if err != nil {
		t.Fatalf("expected Retrieve to succeed, got %v", err)
	}
	if len(res.Docs) != 2 {
		t.Errorf("expected 2 docs, got %d", len(res.Docs))
	}
}
