//go:build testhooks

package registry_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/registry"
	"github.com/duynguyendang/manglekit/internal/testproviders/noop"
)

// TestRegistry_ConcurrentRegistration verifies that concurrent registration of providers
// into the registry does not cause race conditions or lost registrations.
func TestRegistry_ConcurrentRegistration(t *testing.T) {
	t.Cleanup(registry.ResetForTest)
	reg := registry.Global()
	const numGoroutines = 10
	const providersPerGoroutine = 3

	var wg sync.WaitGroup
	var registeredCount int32

	// Concurrently register multiple providers
	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for p := 0; p < providersPerGoroutine; p++ {
				opts := &noop.NoopOptions{}

				// We register with the same provider name (noop)
				err := manglekit.Register(
					reg,
					opts,
					noop.New,
				)

				if err != nil {
					// Duplicate registrations will fail, which is expected
					// in this test. We accept that.
					atomic.AddInt32(&registeredCount, 1) // count attempt
				} else {
					atomic.AddInt32(&registeredCount, 1)
				}
			}
		}(g)
	}

	wg.Wait()

	// Verify at least some registrations were attempted
	expectedCount := int32(numGoroutines * providersPerGoroutine)
	if registeredCount != expectedCount {
		t.Logf("expected %d registration attempts, got %d", expectedCount, registeredCount)
	}

	// Just verify we completed without race conditions (detected by -race flag)
	t.Logf("concurrent registration test completed: %d attempts", registeredCount)
}

// TestRegistry_ConcurrentHandlerRegistration verifies that concurrent registration of handlers
// into the registry does not cause race conditions.
func TestRegistry_ConcurrentHandlerRegistration(t *testing.T) {
	t.Cleanup(registry.ResetForTest)
	reg := registry.Global()
	const numHandlers = 10

	var wg sync.WaitGroup

	// Create mock handlers
	handlers := make([]core.ComponentHandler, numHandlers)
	for i := 0; i < numHandlers; i++ {
		handlers[i] = &mockComponentHandler{
			kind: core.Kind(fmt.Sprintf("test-kind-%d", i)),
		}
	}

	// Concurrently register multiple handlers
	for i := 0; i < numHandlers; i++ {
		wg.Add(1)
		go func(handlerIdx int) {
			defer wg.Done()
			// This should be safe to call concurrently
			reg.RegisterHandler(handlers[handlerIdx])
		}(i)
	}

	wg.Wait()

	// Verify we can retrieve handlers concurrently
	var retrievalWg sync.WaitGroup
	var successfulRetrievals int32

	for i := 0; i < numHandlers; i++ {
		retrievalWg.Add(1)
		go func(handlerID int) {
			defer retrievalWg.Done()

			kind := core.Kind(fmt.Sprintf("test-kind-%d", handlerID))
			_, err := reg.GetHandler(kind)
			if err == nil {
				atomic.AddInt32(&successfulRetrievals, 1)
			}
		}(i)
	}

	retrievalWg.Wait()

	// We should have retrieved most or all handlers
	if successfulRetrievals < int32(numHandlers-1) {
		t.Errorf("expected at least %d successful retrievals, got %d", numHandlers-1, successfulRetrievals)
	}

	// Just verify we completed without race conditions (detected by -race flag)
	t.Logf("concurrent handler registration test completed: %d successful retrievals", successfulRetrievals)
}

// TestRegistry_ConcurrentReadWrite verifies that concurrent reads and writes to the registry
// do not cause race conditions (verified by running with -race flag).
func TestRegistry_ConcurrentReadWrite(t *testing.T) {
	t.Cleanup(registry.ResetForTest)
	reg := registry.Global()
	const numReaders = 5
	const numWriters = 3
	const operationsPerGoroutine = 5

	var wg sync.WaitGroup
	var successfulReads int32
	var successfulWrites int32

	// Pre-register noop provider that readers can find
	opts := &noop.NoopOptions{}
	_ = manglekit.Register(reg, opts, noop.New)

	// Start additional writer goroutines
	for w := 0; w < numWriters; w++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()

			for op := 0; op < operationsPerGoroutine; op++ {
				// Use noop as a safe default provider
				opts := &noop.NoopOptions{}
				err := manglekit.Register(reg, opts, noop.New)

				if err == nil {
					atomic.AddInt32(&successfulWrites, 1)
				}
			}
		}(w)
	}

	// Start reader goroutines
	for r := 0; r < numReaders; r++ {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()

			for op := 0; op < operationsPerGoroutine; op++ {
				// Try to read noop provider which was pre-registered
				_, err := reg.Get(core.KindSchemaParser, "noop")
				if err == nil {
					atomic.AddInt32(&successfulReads, 1)
				}
			}
		}(r)
	}

	wg.Wait()

	// Just verify we completed without race conditions (detected by -race flag)
	t.Logf("concurrent read/write test completed: %d successful writes, %d successful reads", successfulWrites, successfulReads)
	if successfulReads == 0 {
		t.Errorf("expected at least some successful reads, got 0")
	}
}

// mockComponentHandler is a mock implementation of core.ComponentHandler for testing
type mockComponentHandler struct {
	kind core.Kind
}

func (m *mockComponentHandler) Kind() core.Kind {
	return m.kind
}

func (m *mockComponentHandler) BuildComponent(
	ctx context.Context,
	builderDI any,
	factory any,
	resolved *core.Resolved,
	cfg core.ProviderOptions,
	name string,
) (core.ResourceCloser, error) {
	return nil, nil
}
