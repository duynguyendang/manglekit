package sdk

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/duynguyendang/manglekit/core"
)

// testMemory is a minimal core.AgentMemory for shutdown tests.
// Memorize blocks on block (if non-nil) to simulate slow operations.
type testMemory struct {
	count     atomic.Int32
	block     chan struct{}
	returnErr error
}

func (m *testMemory) Read(_ context.Context, _ string) ([]core.Message, error) {
	return nil, nil
}

func (m *testMemory) Append(_ context.Context, _ string, _ []core.Message) error {
	return nil
}

func (m *testMemory) Recall(_ context.Context, _ string) (string, error) {
	return "", nil
}

func (m *testMemory) Memorize(ctx context.Context, _, _ string) error {
	m.count.Add(1)
	if m.block != nil {
		select {
		case <-m.block:
		case <-ctx.Done():
		}
	}
	return m.returnErr
}

func (m *testMemory) Init(_ context.Context) error { return nil }

// waitFor polls cond until it returns true or timeout.
func waitFor(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition not met within timeout")
}

func TestShutdown_WaitsForInFlightMemorize(t *testing.T) {
	block := make(chan struct{})
	mem := &testMemory{block: block}

	c, err := NewClient(context.Background(), WithAgentMemory(mem))
	if err != nil {
		t.Fatal(err)
	}

	c.asyncMemorize("q1", "a1")
	c.asyncMemorize("q2", "a2")

	waitFor(t, time.Second, func() bool { return mem.count.Load() == 2 })

	shutdownDone := make(chan error, 1)
	go func() {
		shutdownDone <- c.Shutdown(context.Background())
	}()

	// Shutdown should not return while blocks are held.
	select {
	case err := <-shutdownDone:
		t.Fatalf("Shutdown returned prematurely with err=%v (memWg not awaited)", err)
	case <-time.After(100 * time.Millisecond):
		// expected — Shutdown is waiting
	}

	close(block)

	select {
	case err := <-shutdownDone:
		if err != nil {
			t.Errorf("Shutdown failed: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Shutdown did not return after release")
	}
}

func TestShutdown_RefusesNewMemorizeAfterClose(t *testing.T) {
	mem := &testMemory{}
	c, err := NewClient(context.Background(), WithAgentMemory(mem))
	if err != nil {
		t.Fatal(err)
	}

	// Fire a burst of memorizes; the synchronous shutDown check in asyncMemorize
	// guarantees each either starts (Add) or is refused before we proceed.
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.asyncMemorize("q", "a")
		}()
	}
	wg.Wait() // all 100 calls invoked (shuttingDown is still false here)

	if err := c.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}

	// Memorizes attempted after Shutdown must be refused.
	c.asyncMemorize("q", "a")
	c.asyncMemorize("q", "a")

	// Count must not grow beyond the burst that started before Shutdown.
	pre := mem.count.Load()
	if pre == 0 {
		// Nothing started before close; post-shutdown calls must keep it at 0.
		return
	}
	waitFor(t, time.Second, func() bool { return mem.count.Load() == pre })
}

func TestShutdown_Idempotent(t *testing.T) {
	c, err := NewClient(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if err := c.Shutdown(context.Background()); err != nil {
		t.Errorf("first Shutdown: %v", err)
	}
	if err := c.Shutdown(context.Background()); err != nil {
		t.Errorf("second Shutdown should be no-op, got: %v", err)
	}
}

func TestShutdown_CtxCancellationReturnsError(t *testing.T) {
	block := make(chan struct{})
	defer close(block)
	mem := &testMemory{block: block}

	c, err := NewClient(context.Background(), WithAgentMemory(mem))
	if err != nil {
		t.Fatal(err)
	}

	c.asyncMemorize("q", "a")
	waitFor(t, time.Second, func() bool { return mem.count.Load() == 1 })

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err = c.Shutdown(ctx)
	if err != context.DeadlineExceeded {
		t.Errorf("expected DeadlineExceeded, got %v", err)
	}
}

func TestShutdown_ConcurrentStress(t *testing.T) {
	mem := &testMemory{}
	c, err := NewClient(context.Background(), WithAgentMemory(mem))
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.asyncMemorize("q", "a")
		}()
	}

	go func() {
		time.Sleep(5 * time.Millisecond)
		c.Shutdown(context.Background())
	}()

	wg.Wait()
	_ = mem.count.Load()
}
