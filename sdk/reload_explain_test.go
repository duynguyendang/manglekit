package sdk

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/duynguyendang/manglekit/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const reloadTestPolicyV1 = `
halt("Req", "denied by v1", "T3").
`

const reloadTestPolicyV2 = `
halt("Req", "denied by v2", "T3").
`

func writePolicy(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "policy.dl")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

func TestClientReloadPolicy_HotSwapAndFailSafe(t *testing.T) {
	ctx := context.Background()
	pathV1 := writePolicy(t, reloadTestPolicyV1)
	pathV2 := writePolicy(t, reloadTestPolicyV2)

	client, err := NewClient(ctx, WithBlueprintPath(pathV1))
	require.NoError(t, err)

	meta := core.ActionMetadata{Name: "test"}
	env := core.NewEnvelope(map[string]string{"action": "test"})

	expectDeny := func(msg string) {
		t.Helper()
		err := client.Engine().Assess(ctx, meta, env)
		require.Error(t, err)
		var alignErr *core.AlignmentError
		require.True(t, errors.As(err, &alignErr))
		assert.Contains(t, alignErr.Message, msg)
	}

	// v1 active.
	expectDeny("denied by v1")

	// Invalid reload: error returned, old policy keeps serving.
	pathBad := writePolicy(t, "halt((((")
	require.Error(t, client.ReloadPolicy(ctx, pathBad))
	expectDeny("denied by v1")

	// Valid reload: new policy serving, old rule gone.
	require.NoError(t, client.ReloadPolicy(ctx, pathV2))
	expectDeny("denied by v2")
}

func TestClientReloadPolicy_ConcurrentGateChecks(t *testing.T) {
	ctx := context.Background()
	pathV1 := writePolicy(t, reloadTestPolicyV1)
	pathV2 := writePolicy(t, reloadTestPolicyV2)

	client, err := NewClient(ctx, WithBlueprintPath(pathV1))
	require.NoError(t, err)

	meta := core.ActionMetadata{Name: "test"}
	env := core.NewEnvelope(map[string]string{"action": "test"})

	var mu sync.Mutex
	var observed []string
	var wg sync.WaitGroup
	stop := make(chan struct{})

	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				err := client.Engine().Assess(ctx, meta, env)
				msg := "<no-deny>"
				if err != nil {
					var alignErr *core.AlignmentError
					if errors.As(err, &alignErr) {
						msg = alignErr.Message
					} else {
						msg = "<engine-error: " + err.Error() + ">"
					}
				}
				mu.Lock()
				observed = append(observed, msg)
				mu.Unlock()
			}
		}()
	}

	for i := 0; i < 25; i++ {
		target := pathV1
		if i%2 == 1 {
			target = pathV2
		}
		require.NoError(t, client.ReloadPolicy(ctx, target))
	}
	close(stop)
	wg.Wait()

	require.NotEmpty(t, observed)
	for _, msg := range observed {
		assert.Contains(t, []string{"denied by v1", "denied by v2"}, msg,
			"gate checks during reload must observe a complete policy (old or new), got %q", msg)
	}
}

func TestClientExplain(t *testing.T) {
	ctx := context.Background()
	path := writePolicy(t, reloadTestPolicyV1)

	client, err := NewClient(ctx, WithBlueprintPath(path))
	require.NoError(t, err)

	expl, err := client.Explain(ctx, `halt("Req", Reason, Tier)`, nil)
	require.NoError(t, err)
	require.True(t, expl.Outcome)
	require.Len(t, expl.Derivations, 1)
	assert.Equal(t, core.TierT3_User, expl.Derivations[0].Tier)
	assert.Contains(t, expl.Derivations[0].Fact, "denied by v1")
}
