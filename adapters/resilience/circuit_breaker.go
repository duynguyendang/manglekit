package resilience

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/duynguyendang/manglekit/core"
)

var ErrCircuitOpen = errors.New("circuit breaker is open")

type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

type CircuitBreakerConfig struct {
	FailureThreshold uint64
	ResetTimeout     time.Duration
}

type CircuitBreaker struct {
	inner      core.Action
	config     CircuitBreakerConfig
	mu         sync.RWMutex
	state      State
	failures   uint64
	lastOpen   time.Time
	generation uint64 // bumped on every state change; protects stale reads
}

// NewCircuitBreaker creates a new CircuitBreaker adapter.
func NewCircuitBreaker(inner core.Action, config CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		inner:  inner,
		config: config,
		state:  StateClosed,
	}
}

// Execute runs the inner action if the circuit is closed or half-open.
func (c *CircuitBreaker) Execute(ctx context.Context, env core.Envelope) (core.Envelope, error) {
	c.mu.RLock()
	state := c.state
	lastOpen := c.lastOpen
	gen := c.generation
	c.mu.RUnlock()

	if state == StateOpen {
		if time.Since(lastOpen) <= c.config.ResetTimeout {
			return core.Envelope{}, ErrCircuitOpen
		}

		// Timeout passed, attempt to transition to HalfOpen.
		// Re-validate under write lock: if the generation moved (e.g. a probe
		// already ran and closed the circuit, or a fresh failure re-opened it),
		// bail out and re-evaluate against the new state.
		c.mu.Lock()
		if c.generation != gen {
			// State was mutated concurrently; restart with fresh snapshot.
			c.mu.Unlock()
			return c.Execute(ctx, env)
		}
		if c.state == StateOpen {
			c.state = StateHalfOpen
			c.generation++
			c.mu.Unlock()
			return c.runProbe(ctx, env)
		}
		// State changed (e.g. to Closed) under us.
		c.mu.Unlock()
		return core.Envelope{}, ErrCircuitOpen
	}

	if state == StateHalfOpen {
		return core.Envelope{}, ErrCircuitOpen
	}

	// StateClosed (or valid enough to try)
	return c.runStandard(ctx, env)
}

func (c *CircuitBreaker) runProbe(ctx context.Context, env core.Envelope) (core.Envelope, error) {
	resp, err := c.inner.Execute(ctx, env)

	c.mu.Lock()
	defer c.mu.Unlock()

	if err != nil {
		// Probe failed, return to Open
		c.state = StateOpen
		c.lastOpen = time.Now()
		c.generation++
	} else {
		// Probe succeeded, close the circuit
		c.state = StateClosed
		c.failures = 0
		c.generation++
	}
	return resp, err
}

func (c *CircuitBreaker) runStandard(ctx context.Context, env core.Envelope) (core.Envelope, error) {
	resp, err := c.inner.Execute(ctx, env)

	c.mu.Lock()
	defer c.mu.Unlock()

	// If the state changed while we were executing (e.g. to Open),
	// we shouldn't modify the state further based on this "stale" execution.
	if c.state != StateClosed {
		return resp, err
	}

	if err != nil {
		c.failures++
		if c.failures >= c.config.FailureThreshold {
			c.state = StateOpen
			c.lastOpen = time.Now()
			c.generation++
		}
	} else {
		c.failures = 0
	}

	return resp, err
}

// Metadata delegates to the inner action's Metadata.
func (c *CircuitBreaker) Metadata() core.ActionMetadata {
	return c.inner.Metadata()
}
