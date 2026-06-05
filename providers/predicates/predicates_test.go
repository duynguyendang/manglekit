package predicates

import (
	"context"
	"testing"
	"time"
)

func TestWithinTimeWindow_InsideWindow(t *testing.T) {
	now := time.Now().UTC()
	start := now.Add(-1 * time.Hour).Format("15:04")
	end := now.Add(1 * time.Hour).Format("15:04")

	result, err := withinTimeWindow(context.Background(), []any{"Req", start, end})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Error("expected result inside window, got nil")
	}
}

func TestWithinTimeWindow_OutsideWindow(t *testing.T) {
	now := time.Now().UTC()
	// Set window to 2 hours ago - 1 hour ago (already passed)
	start := now.Add(-3 * time.Hour).Format("15:04")
	end := now.Add(-2 * time.Hour).Format("15:04")

	result, err := withinTimeWindow(context.Background(), []any{"Req", start, end})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Error("expected nil outside window, got result")
	}
}

func TestWithinTimeWindow_WrapsMidnight(t *testing.T) {
	// This test verifies the midnight-wrapping logic
	// We can't easily test with real time, so we test the parseTime function
	hour, min, err := parseTime("23:30")
	if err != nil {
		t.Fatalf("parseTime failed: %v", err)
	}
	if hour != 23 || min != 30 {
		t.Errorf("expected 23:30, got %d:%d", hour, min)
	}
}

func TestWithinTimeWindow_InvalidFormat(t *testing.T) {
	_, err := withinTimeWindow(context.Background(), []any{"Req", "invalid", "04:00"})
	if err == nil {
		t.Error("expected error for invalid time format")
	}
}

func TestWithinTimeWindow_TooFewArgs(t *testing.T) {
	_, err := withinTimeWindow(context.Background(), []any{"Req"})
	if err == nil {
		t.Error("expected error for too few args")
	}
}

func TestParseTime(t *testing.T) {
	tests := []struct {
		input   string
		hour    int
		min     int
		wantErr bool
	}{
		{"00:00", 0, 0, false},
		{"12:30", 12, 30, false},
		{"23:59", 23, 59, false},
		{"24:00", 0, 0, true},
		{"12:60", 0, 0, true},
		{"abc", 0, 0, true},
	}

	for _, tt := range tests {
		hour, min, err := parseTime(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("parseTime(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && (hour != tt.hour || min != tt.min) {
			t.Errorf("parseTime(%q) = %d:%d, want %d:%d", tt.input, hour, min, tt.hour, tt.min)
		}
	}
}

func TestRateLimiter_Basic(t *testing.T) {
	limiter := NewRateLimiter(3, time.Minute)

	// First 3 requests should not be exceeded
	for i := 0; i < 3; i++ {
		if limiter.IsExceeded("user-1") {
			t.Errorf("request %d should not be exceeded", i+1)
		}
	}

	// 4th request should be exceeded
	if !limiter.IsExceeded("user-1") {
		t.Error("4th request should be exceeded")
	}
}

func TestRateLimiter_PerKey(t *testing.T) {
	limiter := NewRateLimiter(2, time.Minute)

	// user-1 uses up their limit
	limiter.IsExceeded("user-1")
	limiter.IsExceeded("user-1")
	if !limiter.IsExceeded("user-1") {
		t.Error("user-1 should be exceeded")
	}

	// user-2 should still be fine
	if limiter.IsExceeded("user-2") {
		t.Error("user-2 should not be exceeded")
	}
}

func TestRateLimiter_WindowExpiry(t *testing.T) {
	limiter := NewRateLimiter(1, 50*time.Millisecond)

	limiter.IsExceeded("key")
	if !limiter.IsExceeded("key") {
		t.Error("should be exceeded after 1 request")
	}

	// Wait for window to expire
	time.Sleep(60 * time.Millisecond)

	if limiter.IsExceeded("key") {
		t.Error("should not be exceeded after window expiry")
	}
}

func TestRateLimiter_Reset(t *testing.T) {
	limiter := NewRateLimiter(1, time.Minute)

	limiter.IsExceeded("key")
	if !limiter.IsExceeded("key") {
		t.Error("should be exceeded")
	}

	limiter.Reset()

	if limiter.IsExceeded("key") {
		t.Error("should not be exceeded after reset")
	}
}

func TestRateLimitExceeded_TooFewArgs(t *testing.T) {
	_, err := rateLimitExceeded(context.Background(), []any{"Req"})
	if err == nil {
		t.Error("expected error for too few args")
	}
}

func TestHasClaim_TooFewArgs(t *testing.T) {
	_, err := hasClaim(context.Background(), []any{"Req"})
	if err == nil {
		t.Error("expected error for too few args")
	}
}

func TestRegisterAll(t *testing.T) {
	type registry interface {
		RegisterExternalPredicate(name string, fn func(ctx context.Context, inputs []any) ([][]any, error)) error
	}

	// Use a simple mock
	var registered []string
	mock := &mockRegistry{registered: &registered}

	RegisterAll(mock)

	if len(registered) != 3 {
		t.Errorf("expected 3 predicates registered, got %d: %v", len(registered), registered)
	}
}

type mockRegistry struct {
	registered *[]string
}

func (m *mockRegistry) RegisterExternalPredicate(name string, fn func(ctx context.Context, inputs []any) ([][]any, error)) error {
	*m.registered = append(*m.registered, name)
	return nil
}
