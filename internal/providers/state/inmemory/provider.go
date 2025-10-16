package inmemory

import (
	"context"
	"fmt"
	"sync"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/state"
)

func Register(r *manglekit.Registry) {
	r.RegisterStateProvider("inmemory", func(ctx context.Context, opts any, deps manglekit.FactoryDeps) (core.StateProvider, error) {
		var typedOpts state.InMemoryOptions
		if opts != nil {
			if castedOpts, ok := opts.(*state.InMemoryOptions); ok {
				typedOpts = *castedOpts
			} else {
				return nil, fmt.Errorf("invalid options type, expected *state.InMemoryOptions, got %T", opts)
			}
		}
		return New(typedOpts)
	})
	r.RegisterOptions("inmemory", (*state.InMemoryOptions)(nil))
}

// Provider is a thread-safe, in-memory implementation of the core.StateProvider
// interface. It is intended for development, testing, and simple applications
// where state persistence is not required across application restarts.
type Provider struct {
	mu   sync.RWMutex
	data map[string]interface{}
}

// New creates and returns a new in-memory state provider.
// It accepts an Options struct which is currently unused.
func New(opts state.InMemoryOptions) (core.StateProvider, error) {
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
	p.data[sessionID] = state
	return nil
}

// Delete removes the state associated with a given session ID from the in-memory map.
// It is a thread-safe operation. If the session ID does not exist, it does nothing
// and returns nil.
func (p *Provider) Delete(ctx context.Context, sessionID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.data, sessionID)
	return nil
}

// Close is a no-op for the in-memory provider as there are no external
// resources to release. It fulfills the core.ResourceCloser interface.
func (p *Provider) Close(ctx context.Context) error {
	return nil
}
