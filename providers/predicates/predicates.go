// Package predicates provides reference external predicates for use in Datalog policies.
//
// These predicates bridge the gap between Datalog rules and runtime state,
// enabling rules to consult time, rate limits, and identity claims.
//
// Usage:
//
//	engine, _ := engine.New()
//	predicates.RegisterAll(engine)
//
// Then in your Datalog policy:
//
//	rate_limited(Req) :- rate_limit_exceeded(Req, "user-123", 100).
//	time_blocked(Req) :- within_time_window(Req, "02:00", "04:00").
package predicates

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// RateLimitEntry tracks request counts per key within a fixed window.
type RateLimitEntry struct {
	Count     int
	WindowEnd time.Time
}

// RateLimiter implements a simple in-memory fixed-window rate limiter.
// Each key accumulates a count that resets when the window elapses.
type RateLimiter struct {
	mu      sync.Mutex
	entries map[string]*RateLimitEntry
	limit   int
	window  time.Duration
}

// NewRateLimiter creates a new rate limiter.
// limit is the max requests per window. window is the sliding window duration.
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		entries: make(map[string]*RateLimitEntry),
		limit:   limit,
		window:  window,
	}
}

// IsExceeded checks if the key has exceeded the rate limit.
// Returns true if the count exceeds the limit within the window.
func (r *RateLimiter) IsExceeded(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	entry, exists := r.entries[key]

	if !exists || now.After(entry.WindowEnd) {
		// New window
		r.entries[key] = &RateLimitEntry{
			Count:     1,
			WindowEnd: now.Add(r.window),
		}
		return false
	}

	entry.Count++
	return entry.Count > r.limit
}

// Reset clears all entries (useful for testing).
func (r *RateLimiter) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries = make(map[string]*RateLimitEntry)
}

// withinTimeWindow implements: within_time_window(Entity, StartHour, EndHour).
// Returns true if the current time (in UTC) is within the specified hour range.
//
// Example: within_time_window(Req, "02:00", "04:00") is true between 2am and 4am UTC.
func withinTimeWindow(_ context.Context, inputs []any) ([][]any, error) {
	if len(inputs) < 3 {
		return nil, fmt.Errorf("within_time_window requires 3 args: entity, start_hour, end_hour")
	}

	startStr, ok := inputs[1].(string)
	if !ok {
		return nil, fmt.Errorf("start_hour must be string, got %T", inputs[1])
	}
	endStr, ok := inputs[2].(string)
	if !ok {
		return nil, fmt.Errorf("end_hour must be string, got %T", inputs[2])
	}

	startHour, startMin, err := parseTime(startStr)
	if err != nil {
		return nil, fmt.Errorf("invalid start_hour: %w", err)
	}
	endHour, endMin, err := parseTime(endStr)
	if err != nil {
		return nil, fmt.Errorf("invalid end_hour: %w", err)
	}

	now := time.Now().UTC()
	currentMinutes := now.Hour()*60 + now.Minute()
	startMinutes := startHour*60 + startMin
	endMinutes := endHour*60 + endMin

	var inWindow bool
	if startMinutes <= endMinutes {
		inWindow = currentMinutes >= startMinutes && currentMinutes <= endMinutes
	} else {
		// Wraps midnight (e.g., 22:00 to 06:00)
		inWindow = currentMinutes >= startMinutes || currentMinutes <= endMinutes
	}

	if inWindow {
		return [][]any{{true}}, nil
	}
	return nil, nil
}

// rateLimitExceeded implements: rate_limit_exceeded(Entity, Key, Limit).
// Returns true if the key has exceeded the specified limit within a 1-minute window.
//
// Example: rate_limit_exceeded(Req, "user-123", 100) is true if user-123 made >100 requests.
var (
	rateLimitersMu sync.Mutex
	rateLimiters   = make(map[string]*RateLimiter)
	rateCallCount  int
	// lastEvict tracks when the limiter map was last swept so eviction
	// also runs on a time basis (not only every 1000 calls). This bounds
	// memory growth for low-traffic keys that would otherwise linger
	// until the next 1000th call happens to touch the map.
	lastEvict time.Time
)

// evictInterval bounds how often the limiter map is swept for expired
// entries. A call that finds the interval elapsed triggers a sweep; this
// keeps idle limiters from accumulating indefinitely between high-traffic
// windows.
const evictInterval = 30 * time.Second

// evictExpiredLimiters removes limiters whose windows have fully elapsed.
// Callers must hold rateLimitersMu.
func evictExpiredLimiters() {
	now := time.Now()
	for ek, el := range rateLimiters {
		el.mu.Lock()
		expired := true
		for _, e := range el.entries {
			if now.Before(e.WindowEnd) {
				expired = false
				break
			}
		}
		el.mu.Unlock()
		if expired {
			delete(rateLimiters, ek)
		}
	}
	lastEvict = now
}

// getOrCreateLimiter returns a per-key limiter, creating one if needed.
// Keys are formatted as "key:limit" so different limits get separate limiters.
func getOrCreateLimiter(key string, limit int) *RateLimiter {
	rateLimitersMu.Lock()
	defer rateLimitersMu.Unlock()

	k := fmt.Sprintf("%s:%d", key, limit)
	if l, ok := rateLimiters[k]; ok {
		return l
	}
	l := NewRateLimiter(limit, time.Minute)
	rateLimiters[k] = l

	// Periodic eviction: every 1000 calls, or whenever evictInterval has
	// elapsed since the last sweep. The time-based trigger ensures
	// low-traffic keys are reclaimed promptly instead of waiting for the
	// call-count threshold.
	rateCallCount++
	now := time.Now()
	if rateCallCount%1000 == 0 || now.Sub(lastEvict) >= evictInterval {
		evictExpiredLimiters()
	}

	return l
}

func rateLimitExceeded(_ context.Context, inputs []any) ([][]any, error) {
	if len(inputs) < 3 {
		return nil, fmt.Errorf("rate_limit_exceeded requires 3 args: entity, key, limit")
	}

	key, ok := inputs[1].(string)
	if !ok {
		return nil, fmt.Errorf("key must be string, got %T", inputs[1])
	}

	limit := 100 // default
	if limitVal, ok := inputs[2].(int64); ok {
		limit = int(limitVal)
	} else if limitVal, ok := inputs[2].(int); ok {
		limit = limitVal
	}

	limiter := getOrCreateLimiter(key, limit)
	if limiter.IsExceeded(key) {
		return [][]any{{true}}, nil
	}
	return nil, nil
}

// hasClaim is intentionally NOT provided as an external predicate. Claims are
// expressed directly in Datalog as metadata facts:
//
//	has_claim(Req, "email", V) :- meta("claim.email", V).
//
// A previous Go fallback stub silently returned no solutions, which is security
// theater: an external predicate cannot see the entity's facts. Claims must be
// resolved by the engine against the entity's metadata, not by this stub.

// RegisterAll registers all reference external predicates with the given engine.
// This is the recommended way to enable predicates for your policies.
//
// Example:
//
//	eng, _ := engine.New()
//	if err := predicates.RegisterAll(eng); err != nil {
//		log.Fatal(err)
//	}
//
// Parameters:
//   - reg: Anything that has RegisterExternalPredicate(name, fn) method.
//     This includes *engine.PolicyEngine and *engine.MangleRuntime.
//
// Returns an aggregated error if any predicate registration fails; a failure
// must not be silently ignored.
func RegisterAll(reg interface {
	RegisterExternalPredicate(name string, fn func(ctx context.Context, inputs []any) ([][]any, error)) error
}) error {
	var errs []error
	if err := reg.RegisterExternalPredicate("within_time_window", withinTimeWindow); err != nil {
		errs = append(errs, fmt.Errorf("within_time_window: %w", err))
	}
	if err := reg.RegisterExternalPredicate("rate_limit_exceeded", rateLimitExceeded); err != nil {
		errs = append(errs, fmt.Errorf("rate_limit_exceeded: %w", err))
	}
	return errors.Join(errs...)
}

// parseTime parses "HH:MM" format and returns hour, minute.
func parseTime(s string) (int, int, error) {
	var hour, min int
	n, err := fmt.Sscanf(s, "%d:%d", &hour, &min)
	if err != nil || n != 2 {
		return 0, 0, fmt.Errorf("invalid time format %q, expected HH:MM", s)
	}
	if hour < 0 || hour > 23 || min < 0 || min > 59 {
		return 0, 0, fmt.Errorf("time out of range: %s", s)
	}
	return hour, min, nil
}
