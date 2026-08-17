// Package testutil provides deterministic, dependency-free test doubles for
// building examples and tests against manglekit without external services or
// API keys:
//
//   - MockLLM: a scripted core.TextGenerator
//   - DeterministicEmbedder / NewLengthEmbedder / NewKeywordEmbedder:
//     deterministic core.Embedder implementations
//   - InMemoryStateProvider: a core.StateProvider backed by a map
//   - WorkflowSessionStore: an in-memory ports.SessionStateStore
//
// Everything here is deterministic and safe for concurrent use.
package testutil

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/sdk/ports"
)

// Compile-time interface checks.
var (
	_ core.TextGenerator      = (*MockLLM)(nil)
	_ core.Embedder           = (*DeterministicEmbedder)(nil)
	_ core.StateProvider      = (*InMemoryStateProvider)(nil)
	_ ports.SessionStateStore = (*WorkflowSessionStore)(nil)
)

// ===========================================================================
// MockLLM — scripted core.TextGenerator
// ===========================================================================

// MockLLM is a deterministic, scripted implementation of core.TextGenerator.
// Responses are consumed in order (FIFO); once the script is exhausted the
// last response repeats, so a single-response MockLLM never runs dry.
//
//	llm := testutil.NewMockLLM("first answer", "second answer")
//	client.SetLLM(llm)
//
// It also records every prompt it was given (see Prompts) so tests can assert
// on what the supervised action actually sent.
type MockLLM struct {
	mu        sync.Mutex
	responses []string
	prompts   []string
}

// NewMockLLM builds a MockLLM that answers with the given responses in call
// order. With no responses it answers with the empty string.
func NewMockLLM(responses ...string) *MockLLM {
	return &MockLLM{responses: append([]string(nil), responses...)}
}

// next returns the next scripted response (or the last one, or "").
func (m *MockLLM) next(prompt string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.prompts = append(m.prompts, prompt)
	if len(m.responses) == 0 {
		return ""
	}
	if len(m.prompts) > len(m.responses) {
		return m.responses[len(m.responses)-1]
	}
	return m.responses[len(m.prompts)-1]
}

// Complete implements core.TextGenerator.
func (m *MockLLM) Complete(_ context.Context, prompt string) (string, error) {
	return m.next(prompt), nil
}

// Generate implements core.TextGenerator.
func (m *MockLLM) Generate(_ context.Context, prompt string, _ ...core.GenerateOption) (*core.LLMResponse, error) {
	text := m.next(prompt)
	return &core.LLMResponse{
		Text:  text,
		Usage: map[string]int{"prompt": len(prompt), "completion": len(text)},
	}, nil
}

// Stream implements core.TextGenerator: it emits the scripted response as a
// single chunk and closes the channel.
func (m *MockLLM) Stream(ctx context.Context, prompt string) (<-chan core.StreamChunk, error) {
	text := m.next(prompt)
	ch := make(chan core.StreamChunk, 1)
	ch <- core.StreamChunk{Text: text}
	close(ch)
	return ch, nil
}

// Calls returns the number of generation calls made so far.
func (m *MockLLM) Calls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.prompts)
}

// Prompts returns every prompt the mock has been given, in call order.
func (m *MockLLM) Prompts() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string(nil), m.prompts...)
}

// ===========================================================================
// DeterministicEmbedder — deterministic core.Embedder
// ===========================================================================

// DeterministicEmbedder is a core.Embedder whose vectors are a pure function
// of the input text. Construct it with NewLengthEmbedder or
// NewKeywordEmbedder, or directly with a custom VecFor function.
type DeterministicEmbedder struct {
	// Dim is the vector dimension reported by Dimension().
	Dim int
	// VecFor maps text to its vector. It must be pure and deterministic.
	VecFor func(text string) []float32
}

// NewLengthEmbedder returns a 2-dimensional embedder that derives vectors
// from text length — cheap, deterministic, and distinct for different lengths.
func NewLengthEmbedder() *DeterministicEmbedder {
	return &DeterministicEmbedder{
		Dim: 2,
		VecFor: func(text string) []float32 {
			l := float32(len(text))
			return []float32{l / 100.0, float32(int(l)%10) / 10.0}
		},
	}
}

// NewKeywordEmbedder returns an embedder that returns a fixed vector when the
// text contains any of the given keyword substrings, and fallback otherwise.
// keywords maps substring (matched via strings.Contains) to vector; all
// vectors must have the same dimension as fallback.
func NewKeywordEmbedder(keywords map[string][]float32, fallback []float32) *DeterministicEmbedder {
	return &DeterministicEmbedder{
		Dim: len(fallback),
		VecFor: func(text string) []float32 {
			for kw, vec := range keywords {
				if strings.Contains(text, kw) {
					return vec
				}
			}
			return fallback
		},
	}
}

// Embed implements core.Embedder.
func (e *DeterministicEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	return e.VecFor(text), nil
}

// EmbedBatch implements core.Embedder.
func (e *DeterministicEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	vecs := make([][]float32, len(texts))
	for i, t := range texts {
		v, err := e.Embed(ctx, t)
		if err != nil {
			return nil, err
		}
		vecs[i] = v
	}
	return vecs, nil
}

// Dimension implements core.Embedder.
func (e *DeterministicEmbedder) Dimension() int { return e.Dim }

// ===========================================================================
// InMemoryStateProvider — core.StateProvider
// ===========================================================================

// InMemoryStateProvider implements core.StateProvider using a thread-safe map.
// Get marshals the stored state to JSON bytes (nil, nil when absent), Set
// accepts anything JSON-marshalable (plus *core.SessionState / []byte
// shortcuts), and Close resets the store. Suitable for tests and demos; use a
// Badger/Redis-backed provider in production.
type InMemoryStateProvider struct {
	mu    sync.RWMutex
	store map[string]*core.SessionState
}

// NewInMemoryStateProvider creates an empty in-memory state provider.
func NewInMemoryStateProvider() *InMemoryStateProvider {
	return &InMemoryStateProvider{store: make(map[string]*core.SessionState)}
}

// Get implements core.StateProvider: it returns the JSON encoding of the
// session state, or (nil, nil) when the session does not exist.
func (p *InMemoryStateProvider) Get(_ context.Context, sessionID string) (any, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	state, ok := p.store[sessionID]
	if !ok {
		return nil, nil
	}
	data, err := json.Marshal(state)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal state: %w", err)
	}
	return data, nil
}

// Set implements core.StateProvider.
func (p *InMemoryStateProvider) Set(_ context.Context, sessionID string, state any) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var parsed core.SessionState
	switch v := state.(type) {
	case []byte:
		if err := json.Unmarshal(v, &parsed); err != nil {
			return fmt.Errorf("failed to unmarshal state: %w", err)
		}
	case *core.SessionState:
		if v == nil {
			return fmt.Errorf("state cannot be nil *core.SessionState")
		}
		parsed = *v
	case core.SessionState:
		parsed = v
	default:
		data, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("failed to marshal state: %w", err)
		}
		if err := json.Unmarshal(data, &parsed); err != nil {
			return fmt.Errorf("failed to unmarshal state: %w", err)
		}
	}

	p.store[sessionID] = &parsed
	return nil
}

// Delete implements core.StateProvider.
func (p *InMemoryStateProvider) Delete(_ context.Context, sessionID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.store, sessionID)
	return nil
}

// Close implements core.StateProvider: it resets the store.
func (p *InMemoryStateProvider) Close(_ context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.store = make(map[string]*core.SessionState)
	return nil
}

// Size returns the number of stored sessions (test helper).
func (p *InMemoryStateProvider) Size() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.store)
}

// ===========================================================================
// WorkflowSessionStore — ports.SessionStateStore for workflow executors
// ===========================================================================

// WorkflowSessionStore is the thread-safe in-memory ports.SessionStateStore
// used by multiagent.HydratedWorkflowExecutor for workflow-instance
// checkpointing and resume. It re-exports ports.InMemorySessionStore so
// examples and tests need one import for all their test doubles.
type WorkflowSessionStore = ports.InMemorySessionStore

// NewWorkflowSessionStore creates an empty in-memory workflow session store.
func NewWorkflowSessionStore() *WorkflowSessionStore {
	return ports.NewInMemorySessionStore()
}
