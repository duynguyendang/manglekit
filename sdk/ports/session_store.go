package ports

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/duynguyendang/manglekit/core"
)

// InMemorySessionStore is a thread-safe in-memory implementation of SessionStateStore.
// This store is used for transient workflow execution state and does NOT persist to MEB.
type InMemorySessionStore struct {
	mu        sync.RWMutex
	sessions  map[string]*core.WorkflowInstance // Key: sessionID:workflowID
	bySession map[string][]string               // Key: sessionID, Value: list of sessionKeys
}

var _ SessionStateStore = (*InMemorySessionStore)(nil)

// NewInMemorySessionStore creates a new in-memory session store.
func NewInMemorySessionStore() *InMemorySessionStore {
	return &InMemorySessionStore{
		sessions:  make(map[string]*core.WorkflowInstance),
		bySession: make(map[string][]string),
	}
}

func (s *InMemorySessionStore) Create(ctx context.Context, instance *core.WorkflowInstance) error {
	if instance == nil {
		return fmt.Errorf("instance cannot be nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := instance.SessionKey()
	if _, exists := s.sessions[key]; exists {
		return fmt.Errorf("instance %s already exists", key)
	}

	instance.UpdatedAt = time.Now()
	s.sessions[key] = instance
	s.bySession[instance.SessionID] = append(s.bySession[instance.SessionID], key)

	return nil
}

func (s *InMemorySessionStore) Get(ctx context.Context, sessionKey string) (*core.WorkflowInstance, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	instance, exists := s.sessions[sessionKey]
	if !exists {
		return nil, fmt.Errorf("instance %s not found", sessionKey)
	}

	return instance, nil
}

func (s *InMemorySessionStore) Update(ctx context.Context, instance *core.WorkflowInstance) error {
	if instance == nil {
		return fmt.Errorf("instance cannot be nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := instance.SessionKey()
	if _, exists := s.sessions[key]; !exists {
		return fmt.Errorf("instance %s not found", key)
	}

	instance.UpdatedAt = time.Now()
	s.sessions[key] = instance

	return nil
}

func (s *InMemorySessionStore) Delete(ctx context.Context, sessionKey string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	instance, exists := s.sessions[sessionKey]
	if !exists {
		return nil // Already deleted
	}

	delete(s.sessions, sessionKey)

	// Remove from bySession index
	sessionID := instance.SessionID
	if keys, ok := s.bySession[sessionID]; ok {
		var remaining []string
		for _, k := range keys {
			if k != sessionKey {
				remaining = append(remaining, k)
			}
		}
		if len(remaining) == 0 {
			delete(s.bySession, sessionID)
		} else {
			s.bySession[sessionID] = remaining
		}
	}

	return nil
}

func (s *InMemorySessionStore) Exists(ctx context.Context, sessionKey string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, exists := s.sessions[sessionKey]
	return exists
}

func (s *InMemorySessionStore) List(ctx context.Context, sessionID string) ([]*core.WorkflowInstance, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys, ok := s.bySession[sessionID]
	if !ok {
		return []*core.WorkflowInstance{}, nil
	}

	instances := make([]*core.WorkflowInstance, 0, len(keys))
	for _, key := range keys {
		if instance, exists := s.sessions[key]; exists {
			instances = append(instances, instance)
		}
	}

	return instances, nil
}

func (s *InMemorySessionStore) ClearSession(ctx context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	keys, ok := s.bySession[sessionID]
	if !ok {
		return nil
	}

	for _, key := range keys {
		delete(s.sessions, key)
	}

	delete(s.bySession, sessionID)
	return nil
}

// Size returns the number of instances in the store.
func (s *InMemorySessionStore) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.sessions)
}

// SessionCount returns the number of unique sessions.
func (s *InMemorySessionStore) SessionCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.bySession)
}

// NewWorkflowInstance is a convenience function to create a new instance.
func NewWorkflowInstance(workflowID, sessionID string) *core.WorkflowInstance {
	return core.NewWorkflowInstance(workflowID, sessionID)
}
