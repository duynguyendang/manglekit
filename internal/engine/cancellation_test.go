package engine

import (
	"context"
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// heavyProgram builds a runtime with a policy whose evaluation derives a
// large number of facts (pair × num cross products), taking long enough to
// be cancelled mid-evaluation.
func heavyRuntime(t *testing.T, nums int) *MangleRuntime {
	t.Helper()
	rt := NewMangleRuntime()
	ctx := context.Background()
	require.NoError(t, rt.AddPolicy(ctx, `
Decl num(A0).
Decl pair(A0, A1).
Decl triple(A0, A1, A2).
pair(X, Y) :- num(X), num(Y).
triple(X, Y, Z) :- pair(X, Y), num(Z).
`))

	facts := make([]string, nums)
	for i := 0; i < nums; i++ {
		facts[i] = fmt.Sprintf(`num(%d).`, i)
	}
	require.NoError(t, rt.LoadFacts(ctx, facts))
	return rt
}

func TestCooperativeCancellation_CancelledContextStopsEvaluation(t *testing.T) {
	rt := heavyRuntime(t, 70) // 70^3 = 343k derived facts

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := rt.QueryWithSolutions(ctx, nil, `triple(X, Y, Z)`, func(map[string]any) error { return nil })
	elapsed := time.Since(start)

	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Less(t, elapsed, 10*time.Second, "cancelled evaluation must abort promptly")
}

func TestCooperativeCancellation_PreCancelledContext(t *testing.T) {
	rt := heavyRuntime(t, 30)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	rt.mu.Lock()
	before := rt.evalCount
	rt.mu.Unlock()

	err := rt.QueryWithSolutions(ctx, nil, `triple(X, Y, Z)`, func(map[string]any) error { return nil })
	require.ErrorIs(t, err, context.Canceled)

	rt.mu.Lock()
	after := rt.evalCount
	rt.mu.Unlock()
	assert.Equal(t, before, after, "pre-cancelled context must not start evaluation")
}

func TestCooperativeCancellation_NoGoroutineLeakUnderRepeatedTimeouts(t *testing.T) {
	rt := heavyRuntime(t, 40)

	before := runtime.NumGoroutine()
	for i := 0; i < 20; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Millisecond)
		_ = rt.QueryWithSolutions(ctx, nil, `triple(X, Y, Z)`, func(map[string]any) error { return nil })
		cancel()
	}

	// Give any straggler time to surface; the point of cooperative
	// cancellation is that none is spawned at all.
	time.Sleep(200 * time.Millisecond)

	after := runtime.NumGoroutine()
	assert.LessOrEqual(t, after, before+2, "repeated timeouts must not leak evaluation goroutines (before=%d after=%d)", before, after)
}
