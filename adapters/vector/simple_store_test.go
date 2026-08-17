package vector_test

import (
	"context"
	"testing"

	"github.com/duynguyendang/manglekit/adapters/vector"
)

type MockEmbedder struct{}

func (m *MockEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	// Simple deterministic embedding for testing
	// We use 3D vectors.
	// "apple":  [1, 0, 0]
	// "orange": [0.9, 0.1, 0.0]  (Close to apple)
	// "car":    [0, 0, 1]        (Orthogonal to apple)

	switch text {
	case "apple":
		return []float32{1.0, 0.0, 0.0}, nil
	case "orange":
		return []float32{0.9, 0.1, 0.0}, nil
	case "car":
		return []float32{0.0, 0.0, 1.0}, nil
	default:
		return []float32{0.5, 0.5, 0.5}, nil
	}
}

func (m *MockEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	var res [][]float32
	for _, t := range texts {
		e, _ := m.Embed(ctx, t)
		res = append(res, e)
	}
	return res, nil
}

func (m *MockEmbedder) Dimension() int { return 3 }

func TestSimpleStore(t *testing.T) {
	embedder := &MockEmbedder{}
	store := vector.NewSimpleStore(embedder)
	ctx := context.Background()

	// 1. Upsert
	err := store.Upsert(ctx, "1", "apple")
	if err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}
	err = store.Upsert(ctx, "2", "car")
	if err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	// 2. Search "orange" -> should find "apple" (id "1") first because [0.9, 0.1, 0] is closer to [1,0,0] than [0,0,1]
	results, err := store.Search(ctx, "orange", 2)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("Expected results, got empty")
	}

	if results[0] != "1" {
		t.Errorf("Expected top result '1' (apple), got '%s'", results[0])
	}

	// 3. Get
	content, err := store.Get(ctx, "1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if content != "apple" {
		t.Errorf("Expected content 'apple', got '%s'", content)
	}
}
