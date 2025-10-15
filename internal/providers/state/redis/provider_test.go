package redis

import (
	"context"
	"encoding/json"
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
	provider, err := New(ctx, state.RedisOptions{Addr: s.Addr()})
	if err != nil {
		t.Fatalf("failed to create redis state provider: %v", err)
	}
	defer provider.Close(ctx)

	sessionID := "test-session"
	state := map[string]interface{}{"key": "value"}
	stateBytes, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("failed to marshal state: %v", err)
	}

	// Test Set
	if err := provider.Set(ctx, sessionID, stateBytes); err != nil {
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

	var retrievedMap map[string]interface{}
	retrievedBytes, ok := retrievedState.([]byte)
	if !ok {
		t.Fatalf("retrieved state is not []byte, but %T", retrievedState)
	}
	if err := json.Unmarshal(retrievedBytes, &retrievedMap); err != nil {
		t.Fatalf("failed to unmarshal retrieved state: %v", err)
	}

	if val, ok := retrievedMap["key"].(string); !ok || val != "value" {
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
