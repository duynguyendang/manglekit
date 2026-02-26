package memory

import (
	"context"
	"sync"

	"github.com/duynguyendang/manglekit-wip/core"
)

// VolatileStore implements a thread-safe in-memory store.
// It is used for ephemeral storage that does not persist beyond the process or instance.
type VolatileStore struct {
	mu   sync.RWMutex
	data map[string][]core.Message
}

// Read retrieves the chat history for a given session.
func (s *VolatileStore) Read(ctx context.Context, sessionID string) ([]core.Message, error) {
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
	dst := make([]core.Message, len(src))
	copy(dst, src)
	return dst, nil
}

// Append saves the chat history for a given session.
func (s *VolatileStore) Append(ctx context.Context, sessionID string, msgs []core.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.data == nil {
		s.data = make(map[string][]core.Message)
	}

	dst := make([]core.Message, len(msgs))
	copy(dst, msgs)
	s.data[sessionID] = dst
	return nil
}
