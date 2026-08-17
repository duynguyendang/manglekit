package engine

import (
	"time"
)

// The temporal-reasoning methods below delegate PolicyEngine calls to the
// underlying MangleRuntime. Temporal reasoning is experimental and OFF by
// default: call EnableTemporal() before loading any policy to opt in.
// It is typically accessed through the sdk.Client capability surface
// (see sdk/temporal.go).

// EnableTemporal enables temporal reasoning support. It must be called
// before loading any policies.
func (e *PolicyEngine) EnableTemporal() {
	e.runtime.EnableTemporal()
}

// IsTemporalEnabled returns whether temporal reasoning is enabled.
func (e *PolicyEngine) IsTemporalEnabled() bool {
	return e.runtime.IsTemporalEnabled()
}

// AddTemporalFact adds a fact valid during the time range [start, end].
func (e *PolicyEngine) AddTemporalFact(factString string, start, end time.Time) error {
	return e.runtime.AddTemporalFact(factString, start, end)
}

// AddTemporalFactAt adds a fact valid at a single point in time.
func (e *PolicyEngine) AddTemporalFactAt(factString string, at time.Time) error {
	return e.runtime.AddTemporalFactAt(factString, at)
}

// AddTemporalFactInPast adds a fact that was true for a duration ending now.
func (e *PolicyEngine) AddTemporalFactInPast(factString string, duration time.Duration) error {
	return e.runtime.AddTemporalFactInPast(factString, duration)
}

// AddTemporalFactInFuture adds a fact that will be true for a duration from now.
func (e *PolicyEngine) AddTemporalFactInFuture(factString string, duration time.Duration) error {
	return e.runtime.AddTemporalFactInFuture(factString, duration)
}

// AddEternalFact adds a fact that is always true (no temporal bounds).
func (e *PolicyEngine) AddEternalFact(factString string) error {
	return e.runtime.AddEternalFact(factString)
}

// SetEvaluationTime sets the "now" used to evaluate temporal predicates.
func (e *PolicyEngine) SetEvaluationTime(t time.Time) {
	e.runtime.SetEvaluationTime(t)
}

// GetEvaluationTime returns the configured evaluation time.
func (e *PolicyEngine) GetEvaluationTime() time.Time {
	return e.runtime.GetEvaluationTime()
}

// QueryTemporalFactsAt returns facts matching factPattern valid at time at.
func (e *PolicyEngine) QueryTemporalFactsAt(factPattern string, at time.Time) ([]string, error) {
	return e.runtime.QueryTemporalFactsAt(factPattern, at)
}

// QueryTemporalFactsDuring returns facts matching factPattern valid during
// the interval [start, end].
func (e *PolicyEngine) QueryTemporalFactsDuring(factPattern string, start, end time.Time) ([]string, error) {
	return e.runtime.QueryTemporalFactsDuring(factPattern, start, end)
}

// ContainsTemporalFact reports whether a fact is valid at the evaluation time.
func (e *PolicyEngine) ContainsTemporalFact(factString string) (bool, error) {
	return e.runtime.ContainsTemporalFact(factString)
}
