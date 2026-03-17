// Package session provides a transient, in-memory store for coordination facts
// during OODA loop execution. Facts in this store are isolated by SessionID
// and are NOT persisted to MEB (BadgerDB).
package session

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/sdk/ports"
)

type SessionFact struct {
	Subject   string
	Predicate string
	Object    string
	Graph     string
	CreatedAt time.Time
	TTL       time.Duration
}

func (f *SessionFact) IsExpired() bool {
	if f.TTL <= 0 {
		return false
	}
	return time.Since(f.CreatedAt) > f.TTL
}

type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]map[string]*SessionFact // sessionID -> (key -> fact)
	ttl      time.Duration
}

func NewSessionStore() *SessionStore {
	return &SessionStore{
		sessions: make(map[string]map[string]*SessionFact),
		ttl:      10 * time.Minute, // Default TTL: 10 minutes
	}
}

func (s *SessionStore) SetTTL(ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ttl = ttl
}

func (s *SessionStore) Put(ctx context.Context, sessionID, key string, fact *SessionFact) error {
	if sessionID == "" || key == "" {
		return fmt.Errorf("sessionID and key cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.sessions[sessionID] == nil {
		s.sessions[sessionID] = make(map[string]*SessionFact)
	}

	fact.CreatedAt = time.Now()
	if fact.TTL <= 0 {
		fact.TTL = s.ttl
	}

	s.sessions[sessionID][key] = fact
	return nil
}

func (s *SessionStore) Get(ctx context.Context, sessionID, key string) (*SessionFact, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}

	fact, ok := session[key]
	if !ok {
		return nil, fmt.Errorf("fact %s not found in session %s", key, sessionID)
	}

	if fact.IsExpired() {
		return nil, fmt.Errorf("fact %s has expired", key)
	}

	return fact, nil
}

func (s *SessionStore) GetAll(ctx context.Context, sessionID string) ([]*SessionFact, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}

	var facts []*SessionFact
	for _, fact := range session {
		if !fact.IsExpired() {
			facts = append(facts, fact)
		}
	}

	return facts, nil
}

func (s *SessionStore) Delete(ctx context.Context, sessionID, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return nil
	}

	delete(session, key)

	if len(session) == 0 {
		delete(s.sessions, sessionID)
	}

	return nil
}

func (s *SessionStore) ClearSession(ctx context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, sessionID)
	return nil
}

func (s *SessionStore) ListSessions(ctx context.Context) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sessions := make([]string, 0, len(s.sessions))
	for id := range s.sessions {
		sessions = append(sessions, id)
	}

	return sessions, nil
}

func (s *SessionStore) Count(ctx context.Context) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.sessions)
}

func (s *SessionStore) CleanupExpired() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	cleaned := 0
	for sessionID, session := range s.sessions {
		for key, fact := range session {
			if fact.IsExpired() {
				delete(session, key)
				cleaned++
			}
		}
		if len(session) == 0 {
			delete(s.sessions, sessionID)
		}
	}

	return cleaned
}

func (s *SessionStore) ToAtoms(ctx context.Context, sessionID string) ([]core.Atom, error) {
	facts, err := s.GetAll(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	atoms := make([]core.Atom, 0, len(facts))
	for _, f := range facts {
		atoms = append(atoms, core.Atom{
			Subject:   f.Subject,
			Predicate: f.Predicate,
			Object:    f.Object,
		})
	}

	return atoms, nil
}

func (s *SessionStore) ToQuads(ctx context.Context, sessionID string) ([]core.Quad, error) {
	facts, err := s.GetAll(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	quads := make([]core.Quad, 0, len(facts))
	for _, f := range facts {
		quads = append(quads, core.Quad{
			Subject:   f.Subject,
			Predicate: f.Predicate,
			Object:    f.Object,
			Graph:     f.Graph,
		})
	}

	return quads, nil
}

func NewSessionFact(subject, predicate, object, graph string, ttl time.Duration) *SessionFact {
	return &SessionFact{
		Subject:   subject,
		Predicate: predicate,
		Object:    object,
		Graph:     graph,
		TTL:       ttl,
	}
}

type TransientFactsStore struct {
	store *SessionStore
}

func NewTransientFactsStore() *TransientFactsStore {
	return &TransientFactsStore{
		store: NewSessionStore(),
	}
}

var _ ports.TransientStore = (*TransientFactsStore)(nil)

func (t *TransientFactsStore) Put(ctx context.Context, sessionID, key string, fact *ports.TransientFact) error {
	sessionFact := &SessionFact{
		Subject:   fact.Subject,
		Predicate: fact.Predicate,
		Object:    fact.Object,
		Graph:     fact.Graph,
	}
	return t.store.Put(ctx, sessionID, key, sessionFact)
}

func (t *TransientFactsStore) Get(ctx context.Context, sessionID, key string) (*ports.TransientFact, error) {
	fact, err := t.store.Get(ctx, sessionID, key)
	if err != nil {
		return nil, err
	}
	return &ports.TransientFact{
		Subject:   fact.Subject,
		Predicate: fact.Predicate,
		Object:    fact.Object,
		Graph:     fact.Graph,
		CreatedAt: fact.CreatedAt.Format(time.RFC3339),
	}, nil
}

func (t *TransientFactsStore) GetAll(ctx context.Context, sessionID string) ([]*ports.TransientFact, error) {
	facts, err := t.store.GetAll(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	result := make([]*ports.TransientFact, 0, len(facts))
	for _, f := range facts {
		result = append(result, &ports.TransientFact{
			Subject:   f.Subject,
			Predicate: f.Predicate,
			Object:    f.Object,
			Graph:     f.Graph,
			CreatedAt: f.CreatedAt.Format(time.RFC3339),
		})
	}
	return result, nil
}

func (t *TransientFactsStore) Delete(ctx context.Context, sessionID, key string) error {
	return t.store.Delete(ctx, sessionID, key)
}

func (t *TransientFactsStore) ClearSession(ctx context.Context, sessionID string) error {
	return t.store.ClearSession(ctx, sessionID)
}

func (t *TransientFactsStore) ToAtoms(ctx context.Context, sessionID string) ([]core.Atom, error) {
	return t.store.ToAtoms(ctx, sessionID)
}

func (t *TransientFactsStore) ToQuads(ctx context.Context, sessionID string) ([]core.Quad, error) {
	return t.store.ToQuads(ctx, sessionID)
}
