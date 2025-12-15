package inmem

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/duynguyendang/manglekit/config"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/sdk"
)

// InMemoryStore is a naive implementation for testing.
// In a real scenario, this would use embeddings and cosine similarity.
type InMemoryStore struct {
	mu    sync.RWMutex
	facts []string
}

// New creates a new InMemoryStore.
func New(ctx context.Context, cfg config.MemoryConfig) (core.AgentMemory, error) {
	return &InMemoryStore{
		facts: []string{},
	}, nil
}

func (m *InMemoryStore) Init(ctx context.Context) error { return nil }

func (m *InMemoryStore) Recall(ctx context.Context, query string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Naive Keyword Search
	var relevant []string
	tokens := strings.Split(strings.ToLower(query), " ")

	for _, fact := range m.facts {
		score := 0
		for _, token := range tokens {
			if len(token) > 3 && strings.Contains(strings.ToLower(fact), token) {
				score++
			}
		}
		if score > 0 {
			relevant = append(relevant, fact)
		}
	}

	if len(relevant) == 0 {
		return "", nil
	}
	return "Relevant Past Memory:\n- " + strings.Join(relevant, "\n- "), nil
}

func (m *InMemoryStore) Memorize(ctx context.Context, query string, answer string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	fact := fmt.Sprintf("Q: %s | A: %s", query, answer)
	m.facts = append(m.facts, fact)
	return nil
}

// Register installs the plugin
func Register() {
	sdk.RegisterMemoryProvider("inmem", New)
}
