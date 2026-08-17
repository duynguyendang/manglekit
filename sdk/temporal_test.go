package sdk

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTemporalEngine implements the temporalEngine capability interface for
// testing the SDK delegation layer. It embeds MockEvaluator to satisfy
// core.Evaluator.
type mockTemporalEngine struct {
	*MockEvaluator
	enabled bool
}

func (m *mockTemporalEngine) EnableTemporal()         { m.enabled = true }
func (m *mockTemporalEngine) IsTemporalEnabled() bool { return m.enabled }
func (m *mockTemporalEngine) AddTemporalFact(string, time.Time, time.Time) error {
	return nil
}
func (m *mockTemporalEngine) AddTemporalFactAt(string, time.Time) error {
	return nil
}
func (m *mockTemporalEngine) AddTemporalFactInPast(string, time.Duration) error {
	return nil
}
func (m *mockTemporalEngine) AddTemporalFactInFuture(string, time.Duration) error {
	return nil
}
func (m *mockTemporalEngine) AddEternalFact(string) error { return nil }
func (m *mockTemporalEngine) SetEvaluationTime(time.Time) {}
func (m *mockTemporalEngine) GetEvaluationTime() time.Time {
	return time.Time{}
}
func (m *mockTemporalEngine) QueryTemporalFactsAt(string, time.Time) ([]string, error) {
	return nil, nil
}
func (m *mockTemporalEngine) QueryTemporalFactsDuring(string, time.Time, time.Time) ([]string, error) {
	return nil, nil
}
func (m *mockTemporalEngine) ContainsTemporalFact(string) (bool, error) {
	return false, nil
}

func TestTemporal_DelegatesToEngine(t *testing.T) {
	te := &mockTemporalEngine{}
	client, err := NewClient(context.Background(), WithEngine(te))
	require.NoError(t, err)
	defer client.Shutdown(context.Background())

	require.NoError(t, client.EnableTemporal())
	enabled, err := client.IsTemporalEnabled()
	require.NoError(t, err)
	assert.True(t, enabled)

	now := time.Now()
	require.NoError(t, client.AddTemporalFact("permission(user, read)", now.Add(-time.Hour), now.Add(time.Hour)))
	require.NoError(t, client.AddTemporalFactAt("permission(user, read)", now))
	require.NoError(t, client.AddTemporalFactInPast("permission(user, read)", time.Hour))
	require.NoError(t, client.AddTemporalFactInFuture("permission(user, read)", time.Hour))
	require.NoError(t, client.AddEternalFact("god(user)"))
	require.NoError(t, client.SetEvaluationTime(now))
	_, err = client.QueryTemporalFactsAt("permission(_, read)", now)
	require.NoError(t, err)
	_, err = client.QueryTemporalFactsDuring("permission(_, read)", now.Add(-time.Hour), now.Add(time.Hour))
	require.NoError(t, err)
	_, err = client.ContainsTemporalFact("god(user)")
	require.NoError(t, err)
}

func TestTemporal_ErrorsWhenEngineUnsupported(t *testing.T) {
	client, err := NewClient(context.Background(), WithEngine(new(MockEvaluator)))
	require.NoError(t, err)
	defer client.Shutdown(context.Background())

	err = client.EnableTemporal()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not support temporal reasoning")
}
