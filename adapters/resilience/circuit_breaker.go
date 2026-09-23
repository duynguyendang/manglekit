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

	// FailureWhen decides which errors count against provider health. Nil
	// means "count every error" (the historical behavior).
	//
	// Wire a classifier at the boundary that knows the difference, e.g.
	// adapters/ai.IsInfraFailure: without it, a caller-side mistake (bad
	// prompt, unknown model, rejected key) opens the circuit and masks a real
	// outage — and in HALF_OPEN it re-opens the circuit even though the
	// provider demonstrably answered.
	FailureWhen func(error) bool
}

// counts reports whether err should be recorded as an infrastructure failure.
func (c *CircuitBreaker) counts(err error) bool {
	if err == nil {
		return false
	}
	if c.config.FailureWhen == nil {
		return true
	}
	return c.config.FailureWhen(err)
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

	if c.counts(err) {
		// Probe failed with an infrastructure error, return to Open
		c.state = StateOpen
		c.lastOpen = time.Now()
		c.generation++
	} else {
		// The provider ANSWERED — successfully, or with a rejection caused by
		// the request itself. Either way the service is reachable, so the
		// circuit closes; the caller still receives the error unchanged.
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

	switch {
	case err == nil:
		c.failures = 0
	case c.counts(err):
		c.failures++
		if c.failures >= c.config.FailureThreshold {
			c.state = StateOpen
			c.lastOpen = time.Now()
			c.generation++
		}
	default:
		// A caller-side rejection (bad prompt, unknown model, refused
		// credential, safety block) is not evidence the provider is down.
		// Neither credit nor blame: leave the counter where it is.
	}

	return resp, err
}

// Metadata delegates to the inner action's Metadata.
func (c *CircuitBreaker) Metadata() core.ActionMetadata {
	return c.inner.Metadata()
}
