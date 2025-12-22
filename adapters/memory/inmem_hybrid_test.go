package memory

import (
	"context"
	"strings"
	"testing"
)

func TestInMemHybridStore(t *testing.T) {
	store := NewInMemHybridStore()
	ctx := context.Background()

	// Memorize
	err := store.Memorize(ctx, "What is Manglekit?", "A neuro-symbolic AI framework.")
	if err != nil {
		t.Fatalf("Memorize failed: %v", err)
	}
	err = store.Memorize(ctx, "How does it work?", "It combines vector search and logic.")
	if err != nil {
		t.Fatalf("Memorize 2 failed: %v", err)
	}

	// Recall
	result, err := store.Recall(ctx, "Manglekit framework")
	if err != nil {
		t.Fatalf("Recall failed: %v", err)
	}

	if !strings.Contains(result, "neuro-symbolic AI framework") {
		t.Errorf("Recall did not return expected content. Got: %s", result)
	}
}
