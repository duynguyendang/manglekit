package memory_test

import (
	"context"
	"testing"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/engine/memory"
)

func TestVolatileStore(t *testing.T) {
	store := &memory.VolatileStore{}
	ctx := context.Background()

	// Write
	msgs := []core.Message{{Role: "user", Content: "hello"}}
	if err := store.Write(ctx, "session1", msgs); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Read
	readMsgs, err := store.Read(ctx, "session1")
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if len(readMsgs) != 1 {
		t.Errorf("Expected 1 msg, got %d", len(readMsgs))
	}
	if readMsgs[0].Content != "hello" {
		t.Errorf("Content mismatch")
	}

	// Isolation
	readMsgs2, _ := store.Read(ctx, "session2")
	if len(readMsgs2) != 0 {
		t.Errorf("Expected 0 msgs for session2")
	}
}
