package inmemory

import (
	"context"
	"fmt"
	"sync"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
)

var ErrProviderClosed = fmt.Errorf("provider is closed")

type Options struct {
	ContextWindow int
}

func (o *Options) ProviderName() string { return "in-memory" }
func (o *Options) ProviderKind() core.Kind   { return core.KindStateProvider }
func (o *Options) GetProviderOptions() any { return o }

func Register(r *manglekit.Registry) {
	manglekit.Register(r, &Options{},
		func(ctx context.Context, deps diapi.NoopDeps, cfg *Options) (core.StateProvider, error) {
			return New(*cfg)
		},
	)
}

// Provider is a thread-safe, in-memory implementation of the core.StateProvider
// interface. It is intended for development, testing, and simple applications
// where state persistence is not required across application restarts.
type Provider struct {
	mu     sync.RWMutex
	data   map[string]interface{}
	closed bool
}

// New creates and returns a new in-memory state provider.
// It accepts an Options struct which is currently unused.
func New(opts Options) (core.StateProvider, error) {
	return &Provider{
		data: make(map[string]interface{}),
	}, nil
}

// Get retrieves the state for a given session ID from the in-memory map.
// It is a thread-safe operation. If no state is found for the session ID,
// it returns (nil, nil) as per the core.StateProvider interface contract.
func (p *Provider) Get(ctx context.Context, sessionID string) (interface{}, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.closed {
		return nil, ErrProviderClosed
	}
	state, ok := p.data[sessionID]
	if !ok {
		return nil, nil
	}
	return state, nil
}

// Set saves or overwrites the state for a given session ID in the in-memory map.
// It is a thread-safe operation.
func (p *Provider) Set(ctx context.Context, sessionID string, state interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return ErrProviderClosed
	}
	p.data[sessionID] = state
	return nil
}

// Delete removes the state associated with a given session ID from the in-memory map.
// It is a thread-safe operation. If the session ID does not exist, it does nothing
// and returns nil.
func (p *Provider) Delete(ctx context.Context, sessionID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return ErrProviderClosed
	}
	delete(p.data, sessionID)
	return nil
}

// Close is a no-op for the in-memory provider as there are no external
// resources to release. It fulfills the core.ResourceCloser interface.
func (p *Provider) Close(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closed = true
	return nil
}
