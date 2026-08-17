package sdk

import (
	"fmt"
	"time"
)

// temporalEngine is the capability a policy engine must implement to support
// temporal reasoning (implemented by *engine.PolicyEngine). Temporal reasoning
// is EXPERIMENTAL and OFF by default — call EnableTemporal before loading any
// policy to opt in. It is intended for time-travel / compliance queries where
// facts carry validity windows.
type temporalEngine interface {
	EnableTemporal()
	IsTemporalEnabled() bool
	AddTemporalFact(factString string, start, end time.Time) error
	AddTemporalFactAt(factString string, at time.Time) error
	AddTemporalFactInPast(factString string, duration time.Duration) error
	AddTemporalFactInFuture(factString string, duration time.Duration) error
	AddEternalFact(factString string) error
	SetEvaluationTime(t time.Time)
	GetEvaluationTime() time.Time
	QueryTemporalFactsAt(factPattern string, at time.Time) ([]string, error)
	QueryTemporalFactsDuring(factPattern string, start, end time.Time) ([]string, error)
	ContainsTemporalFact(factString string) (bool, error)
}

// temporalEngineOrNil resolves the engine's temporalEngine capability, or
// nil when the engine does not support temporal reasoning.
func (c *Client) temporalEngineOrNil() temporalEngine {
	if c.engine == nil {
		return nil
	}
	te, ok := c.engine.(temporalEngine)
	if !ok {
		return nil
	}
	return te
}

// EnableTemporal enables temporal reasoning on the policy engine. It must be
// called before loading any policies. Returns an error if the engine does not
// support temporal reasoning.
func (c *Client) EnableTemporal() error {
	te := c.temporalEngineOrNil()
	if te == nil {
		return fmt.Errorf("engine %T does not support temporal reasoning", c.engine)
	}
	te.EnableTemporal()
	return nil
}

// IsTemporalEnabled reports whether temporal reasoning is enabled.
func (c *Client) IsTemporalEnabled() (bool, error) {
	te := c.temporalEngineOrNil()
	if te == nil {
		return false, fmt.Errorf("engine %T does not support temporal reasoning", c.engine)
	}
	return te.IsTemporalEnabled(), nil
}

// AddTemporalFact records a fact valid during the time range [start, end].
func (c *Client) AddTemporalFact(factString string, start, end time.Time) error {
	te := c.temporalEngineOrNil()
	if te == nil {
		return fmt.Errorf("engine %T does not support temporal reasoning", c.engine)
	}
	return te.AddTemporalFact(factString, start, end)
}

// AddTemporalFactAt records a fact valid at a single point in time.
func (c *Client) AddTemporalFactAt(factString string, at time.Time) error {
	te := c.temporalEngineOrNil()
	if te == nil {
		return fmt.Errorf("engine %T does not support temporal reasoning", c.engine)
	}
	return te.AddTemporalFactAt(factString, at)
}

// AddTemporalFactInPast records a fact that was true for a duration ending now.
func (c *Client) AddTemporalFactInPast(factString string, duration time.Duration) error {
	te := c.temporalEngineOrNil()
	if te == nil {
		return fmt.Errorf("engine %T does not support temporal reasoning", c.engine)
	}
	return te.AddTemporalFactInPast(factString, duration)
}

// AddTemporalFactInFuture records a fact that will be true for a duration from now.
func (c *Client) AddTemporalFactInFuture(factString string, duration time.Duration) error {
	te := c.temporalEngineOrNil()
	if te == nil {
		return fmt.Errorf("engine %T does not support temporal reasoning", c.engine)
	}
	return te.AddTemporalFactInFuture(factString, duration)
}

// AddEternalFact records a fact that is always true (no temporal bounds).
func (c *Client) AddEternalFact(factString string) error {
	te := c.temporalEngineOrNil()
	if te == nil {
		return fmt.Errorf("engine %T does not support temporal reasoning", c.engine)
	}
	return te.AddEternalFact(factString)
}

// SetEvaluationTime sets the "now" used to evaluate temporal predicates.
func (c *Client) SetEvaluationTime(t time.Time) error {
	te := c.temporalEngineOrNil()
	if te == nil {
		return fmt.Errorf("engine %T does not support temporal reasoning", c.engine)
	}
	te.SetEvaluationTime(t)
	return nil
}

// QueryTemporalFactsAt returns facts matching factPattern valid at time at.
func (c *Client) QueryTemporalFactsAt(factPattern string, at time.Time) ([]string, error) {
	te := c.temporalEngineOrNil()
	if te == nil {
		return nil, fmt.Errorf("engine %T does not support temporal reasoning", c.engine)
	}
	return te.QueryTemporalFactsAt(factPattern, at)
}

// QueryTemporalFactsDuring returns facts matching factPattern valid during
// the interval [start, end].
func (c *Client) QueryTemporalFactsDuring(factPattern string, start, end time.Time) ([]string, error) {
	te := c.temporalEngineOrNil()
	if te == nil {
		return nil, fmt.Errorf("engine %T does not support temporal reasoning", c.engine)
	}
	return te.QueryTemporalFactsDuring(factPattern, start, end)
}

// ContainsTemporalFact reports whether a fact is valid at the evaluation time.
func (c *Client) ContainsTemporalFact(factString string) (bool, error) {
	te := c.temporalEngineOrNil()
	if te == nil {
		return false, fmt.Errorf("engine %T does not support temporal reasoning", c.engine)
	}
	return te.ContainsTemporalFact(factString)
}
