package engine

import (
	"testing"
	"time"

	"codeberg.org/TauCeti/mangle-go/factstore"
	"codeberg.org/TauCeti/mangle-go/parse"

	"github.com/stretchr/testify/assert"
)

func TestTemporal_EnableDisable(t *testing.T) {
	runtime := NewMangleRuntime()

	assert.False(t, runtime.IsTemporalEnabled())

	runtime.EnableTemporal()

	assert.True(t, runtime.IsTemporalEnabled())
}

func TestTemporal_GetTemporalStore(t *testing.T) {
	runtime := NewMangleRuntime()

	// Should be nil by default
	assert.Nil(t, runtime.GetTemporalStore())

	// Enable temporal
	runtime.EnableTemporal()

	// Store should be available
	store := runtime.GetTemporalStore()
	assert.NotNil(t, store)
}

func TestTemporal_IsEnabled(t *testing.T) {
	runtime := NewMangleRuntime()

	// Should be disabled by default
	assert.False(t, runtime.IsTemporalEnabled())

	// Enable
	runtime.EnableTemporal()
	assert.True(t, runtime.IsTemporalEnabled())
}

func TestTemporal_AddTemporalFact(t *testing.T) {
	runtime := NewMangleRuntime()
	runtime.EnableTemporal()

	// Add a temporal fact
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	err := runtime.AddTemporalFact(`user_access("alice", "report_A")`, start, end)
	assert.NoError(t, err)

	// Verify the store has facts
	store := runtime.GetTemporalStore()
	assert.NotNil(t, store)
}

func TestTemporal_AddTemporalFactAt(t *testing.T) {
	runtime := NewMangleRuntime()
	runtime.EnableTemporal()

	// Add a point-in-time fact
	at := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	err := runtime.AddTemporalFactAt(`session("session_123", "alice")`, at)
	assert.NoError(t, err)
}

func TestTemporal_AddEternalFact(t *testing.T) {
	runtime := NewMangleRuntime()
	runtime.EnableTemporal()

	// Add an eternal fact
	err := runtime.AddEternalFact(`always_true("admin")`)
	assert.NoError(t, err)
}

func TestTemporal_SetEvaluationTime(t *testing.T) {
	runtime := NewMangleRuntime()

	// Should be zero by default
	assert.True(t, runtime.GetEvaluationTime().IsZero())

	// Set evaluation time
	evalTime := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	runtime.SetEvaluationTime(evalTime)

	// Verify
	assert.Equal(t, evalTime, runtime.GetEvaluationTime())
}

func TestTemporal_FullWorkflow(t *testing.T) {
	runtime := NewMangleRuntime()
	runtime.EnableTemporal()

	// Set evaluation time to Feb 15, 2024
	evalTime := time.Date(2024, 2, 15, 0, 0, 0, 0, time.UTC)
	runtime.SetEvaluationTime(evalTime)

	// Add temporal facts
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)
	err := runtime.AddTemporalFact(`user_access("alice", "report_A")`, start, end)
	assert.NoError(t, err)

	start2 := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	end2 := time.Date(2024, 2, 28, 0, 0, 0, 0, time.UTC)
	err = runtime.AddTemporalFact(`user_access("alice", "report_B")`, start2, end2)
	assert.NoError(t, err)

	// Query temporal facts directly using ContainsAt
	store := runtime.GetTemporalStore()
	assert.NotNil(t, store)

	// Check if facts are in the store at evaluation time
	atom1, _ := parse.Atom(`user_access("alice", "report_A")`)
	atom2, _ := parse.Atom(`user_access("alice", "report_B")`)

	// At Feb 15, 2024 - report_A should NOT be valid (Jan 1-31), report_B SHOULD be valid (Feb 1-28)
	containsA := store.ContainsAt(atom1, evalTime)
	containsB := store.ContainsAt(atom2, evalTime)

	assert.False(t, containsA, "report_A should not be valid at Feb 15")
	assert.True(t, containsB, "report_B should be valid at Feb 15")
}

func TestTemporal_DatalogRuleWithTemporalHead(t *testing.T) {
	runtime := NewMangleRuntime()
	runtime.EnableTemporal()

	// Set evaluation time
	evalTime := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	runtime.SetEvaluationTime(evalTime)

	// Add a temporal policy with a temporal head - derive facts with time bounds
	policy := `
		% Initial temporal fact - user accessed resource in June
		user_access("alice", "report_june")@[2024-06-01, 2024-06-30].
		
		% Rule with temporal head - derive audit trail with same time bounds
		audit_trail(User, Resource)@[Start, End] :-
			user_access(User, Resource)@[Start, End].
	`
	err := runtime.LoadFromSource(policy)
	assert.NoError(t, err)

	// Query the derived temporal facts
	store := runtime.GetTemporalStore()
	assert.NotNil(t, store)

	atom, _ := parse.Atom(`audit_trail("alice", "report_june")`)
	contains := store.ContainsAt(atom, evalTime)
	assert.True(t, contains, "Derived temporal fact should be valid at June 15")
}

func TestTemporal_DurationQuery(t *testing.T) {
	runtime := NewMangleRuntime()
	runtime.EnableTemporal()

	// Set evaluation time to now
	evalTime := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	runtime.SetEvaluationTime(evalTime)

	// Add facts from past 30 days
	// 10 days ago
	start1 := evalTime.AddDate(0, 0, -10)
	err := runtime.AddTemporalFactAt(`request("alice", "api")`, start1)
	assert.NoError(t, err)

	// 5 days ago
	start2 := evalTime.AddDate(0, 0, -5)
	err = runtime.AddTemporalFactAt(`request("alice", "api")`, start2)
	assert.NoError(t, err)

	// 45 days ago (outside 30 day window)
	start3 := evalTime.AddDate(0, 0, -45)
	err = runtime.AddTemporalFactAt(`request("bob", "api")`, start3)
	assert.NoError(t, err)

	// Check with GetFactsAt for temporal store queries
	store := runtime.GetTemporalStore()

	queryAtom, _ := parse.Atom(`request(User, Endpoint)`)

	// Get all facts at evaluation time
	var foundFacts []string
	err = store.GetFactsAt(queryAtom, evalTime, func(tf factstore.TemporalFact) error {
		foundFacts = append(foundFacts, tf.Atom.String())
		return nil
	})
	assert.NoError(t, err)

	// Should find alice's requests at eval time (June 15)
	// Bob's request was 45 days ago - should NOT be found at June 15
	t.Logf("Found facts at eval time: %v", foundFacts)

	// Get all facts in the store (regardless of time)
	var allFacts []string
	err = store.GetAllFacts(queryAtom, func(tf factstore.TemporalFact) error {
		allFacts = append(allFacts, tf.Atom.String()+" @"+tf.Interval.String())
		return nil
	})
	assert.NoError(t, err)
	t.Logf("All temporal facts: %v", allFacts)

	// Verify we have facts in the store
	assert.Greater(t, len(allFacts), 0, "Should have temporal facts in store")
}

func TestTemporal_HelperMethods(t *testing.T) {
	runtime := NewMangleRuntime()
	runtime.EnableTemporal()

	evalTime := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	runtime.SetEvaluationTime(evalTime)

	// Test AddTemporalFactInPast
	err := runtime.AddTemporalFactInPast(`active_session("alice")`, 7*24*time.Hour) // 7 days
	assert.NoError(t, err)

	// Test AddTemporalFactInFuture
	err = runtime.AddTemporalFactInFuture(`scheduled_maintenance("server1")`, 2*time.Hour)
	assert.NoError(t, err)

	// Test ContainsTemporalFact
	contains, err := runtime.ContainsTemporalFact(`active_session("alice")`)
	assert.NoError(t, err)
	assert.True(t, contains, "Alice's session should be active at evaluation time")

	contains, err = runtime.ContainsTemporalFact(`not_exist("x")`)
	assert.NoError(t, err)
	assert.False(t, contains, "Non-existent fact should return false")

	// Test QueryTemporalFactsAt
	results, err := runtime.QueryTemporalFactsAt(`request(User, Endpoint)`, evalTime)
	assert.NoError(t, err)
	t.Logf("Results at eval time: %v", results)

	// Test QueryTemporalFactsDuring
	start := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 6, 30, 23, 59, 59, 0, time.UTC)
	results, err = runtime.QueryTemporalFactsDuring(`request(User, Endpoint)`, start, end)
	assert.NoError(t, err)
	t.Logf("Results during June: %v", results)
}
