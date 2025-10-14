package inmemory

import (
	"context"
	"testing"

	"github.com/duynguyendang/manglekit/state"
)

func TestInMemoryStateProvider(t *testing.T) {
	ctx := context.Background()
	provider, err := New(state.InMemoryOptions{})
	if err != nil {
		t.Fatalf("failed to create in-memory state provider: %v", err)
	}

	sessionID := "test-session"
	state := map[string]interface{}{"key": "value"}

	// Test Set
	if err := provider.Set(ctx, sessionID, state); err != nil {
		t.Errorf("Set failed: %v", err)
	}

	// Test Get
	retrievedState, err := provider.Get(ctx, sessionID)
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}
	if retrievedState == nil {
		t.Errorf("Get returned nil state")
	}
	retrievedMap, ok := retrievedState.(map[string]interface{})
	if !ok {
		t.Errorf("retrieved state is not a map")
	}
	if retrievedMap["key"] != "value" {
		t.Errorf("retrieved state has wrong value")
	}

	// Test Delete
	if err := provider.Delete(ctx, sessionID); err != nil {
		t.Errorf("Delete failed: %v", err)
	}

	// Test Get after Delete
	retrievedState, err = provider.Get(ctx, sessionID)
	if err != nil {
		t.Errorf("Get after Delete failed: %v", err)
	}
	if retrievedState != nil {
		t.Errorf("Get after Delete returned non-nil state")
	}
}
