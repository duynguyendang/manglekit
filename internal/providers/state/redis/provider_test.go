package redis

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/duynguyendang/manglekit/state"
)

func TestRedisStateProvider(t *testing.T) {
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer s.Close()

	ctx := context.Background()
	provider, err := New(state.RedisOptions{Addr: s.Addr()})
	if err != nil {
		t.Fatalf("failed to create redis state provider: %v", err)
	}
	defer provider.Close(ctx)

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
		t.Errorf("retrieved state is not a map, but %T", retrievedState)
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
