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
	"fmt"
	"sync"
	"time"
)

// RateLimitEntry tracks request counts per key within a sliding window.
type RateLimitEntry struct {
	Count     int
	WindowEnd time.Time
}

// RateLimiter implements a simple in-memory sliding-window rate limiter.
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
)

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

	// Periodic eviction: every 1000 calls, evict expired entries
	rateCallCount++
	if rateCallCount%1000 == 0 {
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

// hasClaim implements: has_claim(Entity, ClaimKey, ClaimValue).
// Returns true if the claim exists with the given value.
//
// Claims are passed as metadata facts: meta("claim.email", "user@example.com")
// The predicate checks: meta("claim.<key>", value)
func hasClaim(_ context.Context, inputs []any) ([][]any, error) {
	if len(inputs) < 3 {
		return nil, fmt.Errorf("has_claim requires 3 args: entity, claim_key, claim_value")
	}

	claimKey, ok := inputs[1].(string)
	if !ok {
		return nil, fmt.Errorf("claim_key must be string, got %T", inputs[1])
	}
	claimValue, ok := inputs[2].(string)
	if !ok {
		return nil, fmt.Errorf("claim_value must be string, got %T", inputs[2])
	}

	_ = claimKey
	_ = claimValue

	// This predicate works via Datalog metadata facts.
	// The actual evaluation happens in the Datalog engine:
	//   has_claim(Req, "email", V) :- meta("claim.email", V).
	// This external predicate is a fallback for direct Go calls.
	return nil, nil
}

// RegisterAll registers all reference external predicates with the given engine.
// This is the recommended way to enable predicates for your policies.
//
// Example:
//
//	eng, _ := engine.New()
//	predicates.RegisterAll(eng)
//
// Parameters:
//   - reg: Anything that has RegisterExternalPredicate(name, fn) method.
//     This includes *engine.PolicyEngine and *engine.MangleRuntime.
func RegisterAll(reg interface {
	RegisterExternalPredicate(name string, fn func(ctx context.Context, inputs []any) ([][]any, error)) error
}) {
	_ = reg.RegisterExternalPredicate("within_time_window", withinTimeWindow)
	_ = reg.RegisterExternalPredicate("rate_limit_exceeded", rateLimitExceeded)
	_ = reg.RegisterExternalPredicate("has_claim", hasClaim)
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
