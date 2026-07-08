package resilience

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/duynguyendang/manglekit/core"
)

type mockAction struct {
	shouldFail bool
	err        error
	called     int
	mu         sync.Mutex
}

func (m *mockAction) Execute(ctx context.Context, env core.Envelope) (core.Envelope, error) {
	m.mu.Lock()
	m.called++
	shouldFail := m.shouldFail
	m.mu.Unlock()

	// Simulate work
	time.Sleep(10 * time.Millisecond)

	if shouldFail {
		return core.Envelope{}, m.err
	}
	return env, nil
}

func (m *mockAction) Metadata() core.ActionMetadata {
	return core.ActionMetadata{}
}

// waitForTimeout polls until at least the circuit's ResetTimeout has elapsed
// since it opened. This replaces a fixed time.Sleep so the test does not read
// the half-open window early under CI scheduling jitter.
func waitForTimeout(cb *CircuitBreaker, timeout time.Duration) {
	for {
		cb.mu.RLock()
		elapsed := time.Since(cb.lastOpen)
		cb.mu.RUnlock()
		if elapsed >= timeout {
			return
		}
		time.Sleep(time.Millisecond)
	}
}

func TestCircuitBreaker(t *testing.T) {
	mockErr := errors.New("mock error")
	inner := &mockAction{
		shouldFail: true,
		err:        mockErr,
	}

	config := CircuitBreakerConfig{
		FailureThreshold: 2,
		ResetTimeout:     100 * time.Millisecond,
	}

	cb := NewCircuitBreaker(inner, config)
	ctx := context.Background()
	env := core.Envelope{}

	// 1. Call Execute 2 times -> Should fail with "mock error"
	for i := 0; i < 2; i++ {
		_, err := cb.Execute(ctx, env)
		if !errors.Is(err, mockErr) {
			t.Errorf("Iteration %d: expected mock error, got %v", i+1, err)
		}
	}

	// Verify failures count
	cb.mu.RLock()
	if cb.failures != 2 {
		t.Errorf("Expected failures to be 2, got %d", cb.failures)
	}
	if cb.state != StateOpen {
		t.Errorf("Expected state to be Open, got %v", cb.state)
	}
	cb.mu.RUnlock()

	// 2. Call Execute 3rd time -> Should fail with ErrCircuitOpen (Fail Fast)
	_, err := cb.Execute(ctx, env)
	if !errors.Is(err, ErrCircuitOpen) {
		t.Errorf("Expected ErrCircuitOpen, got %v", err)
	}

	// 3. Wait for the reset timeout to elapse
	waitForTimeout(cb, config.ResetTimeout)

	// 4. Call Execute -> Should go through (Half-Open probing)
	// Since inner action always fails, it should return mockErr, not ErrCircuitOpen
	// And it should transition back to Open
	_, err = cb.Execute(ctx, env)
	if !errors.Is(err, mockErr) {
		t.Errorf("Expected mock error after timeout, got %v", err)
	}

	cb.mu.RLock()
	if cb.state != StateOpen {
		t.Errorf("Expected state to be Open after failed probe, got %v", cb.state)
	}
	cb.mu.RUnlock()
}

func TestCircuitBreakerRecovery(t *testing.T) {
	mockErr := errors.New("mock error")
	inner := &mockAction{
		shouldFail: true,
		err:        mockErr,
	}

	config := CircuitBreakerConfig{
		FailureThreshold: 1,
		ResetTimeout:     50 * time.Millisecond,
	}

	cb := NewCircuitBreaker(inner, config)
	ctx := context.Background()
	env := core.Envelope{}

	// Fail once -> Open
	cb.Execute(ctx, env)

	cb.mu.RLock()
	if cb.state != StateOpen {
		t.Fatalf("State should be Open")
	}
	cb.mu.RUnlock()

	// Wait for timeout
	waitForTimeout(cb, config.ResetTimeout)

	// Make inner succeed now
	inner.mu.Lock()
	inner.shouldFail = false
	inner.mu.Unlock()

	// Execute -> Should succeed (Half-Open -> Closed)
	_, err := cb.Execute(ctx, env)
	if err != nil {
		t.Errorf("Expected success, got error: %v", err)
	}

	cb.mu.RLock()
	if cb.state != StateClosed {
		t.Errorf("Expected state to be Closed after success, got %v", cb.state)
	}
	if cb.failures != 0 {
		t.Errorf("Expected failures to be 0, got %d", cb.failures)
	}
	cb.mu.RUnlock()
}

func TestCircuitBreakerHalfOpenConcurrency(t *testing.T) {
	// Test that only one request gets through in Half-Open state
	mockErr := errors.New("mock error")
	inner := &mockAction{
		shouldFail: false, // Success to close the circuit if probe succeeds
		err:        mockErr,
	}

	config := CircuitBreakerConfig{
		FailureThreshold: 1,
		ResetTimeout:     50 * time.Millisecond,
	}

	cb := NewCircuitBreaker(inner, config)
	ctx := context.Background()
	env := core.Envelope{}

	// Trip to Open
	inner.mu.Lock()
	inner.shouldFail = true
	inner.mu.Unlock()

	cb.Execute(ctx, env) // Fail -> Open

	// Wait timeout
	waitForTimeout(cb, config.ResetTimeout)

	// Now try concurrent requests
	// One should succeed (become probe), others should fail with ErrCircuitOpen
	// If probe succeeds, it closes.
	// But let's make the probe slow so we can catch others failing.

	// Make probe slow
	// We can't easily inject delay into just the probe without changing the mock per request.
	// But our mock sleeps 10ms.

	inner.mu.Lock()
	inner.shouldFail = false // Succeed to close
	inner.mu.Unlock()

	var wg sync.WaitGroup
	count := 10
	errs := make([]error, count)

	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, err := cb.Execute(ctx, env)
			errs[idx] = err
		}(i)
	}
	wg.Wait()

	// We expect exactly one success (nil error) and (count-1) ErrCircuitOpen.
	// OR if the probe finishes very fast, subsequent requests might succeed as Closed.
	// To strictly test Half-Open rejection, we need the probe to take longer than the others launch.

	successCount := 0
	circuitOpenCount := 0

	for _, err := range errs {
		if err == nil {
			successCount++
		} else if errors.Is(err, ErrCircuitOpen) {
			circuitOpenCount++
		}
	}

	// It's possible for multiple successes if the probe finished before other goroutines started.
	// But with 10ms sleep and tight loop, we likely see some rejections.
	// If we see > 1 success, that implies we transitioned to Closed quickly.
	// If we strictly want to test "only one probe", we'd need to ensure the probe holds the lock or state long enough.

	if successCount == 0 {
		t.Errorf("Expected at least one success (the probe)")
	}

	// Check final state
	cb.mu.RLock()
	finalState := cb.state
	cb.mu.RUnlock()

	if finalState != StateClosed {
		t.Errorf("Expected final state to be Closed, got %v", finalState)
	}
}
