package memory

import (
	"context"
	"sync"

	"github.com/duynguyendang/manglekit/core"
)

// VolatileStore implements a thread-safe in-memory store.
// It is used for ephemeral storage that does not persist beyond the process or instance.
type VolatileStore struct {
	mu   sync.RWMutex
	data map[string][]core.ChatMessage
}

// Read retrieves the chat history for a given session.
func (s *VolatileStore) Read(ctx context.Context, sessionID string) ([]core.ChatMessage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.data == nil {
		return nil, nil
	}
	// Return copy
	src, ok := s.data[sessionID]
	if !ok {
		return nil, nil
	}
	dst := make([]core.ChatMessage, len(src))
	copy(dst, src)
	return dst, nil
}

// Write saves the chat history for a given session.
func (s *VolatileStore) Write(ctx context.Context, sessionID string, msgs []core.ChatMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.data == nil {
		s.data = make(map[string][]core.ChatMessage)
	}

	dst := make([]core.ChatMessage, len(msgs))
	copy(dst, msgs)
	s.data[sessionID] = dst
	return nil
}
