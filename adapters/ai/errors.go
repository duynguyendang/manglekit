package ai

import (
	"errors"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core/status"
)

// ErrModelBlocked re-exports Genkit's safety-refusal sentinel so callers can
// branch on "the provider refused" without importing Genkit. The value is the
// same sentinel, so errors.Is works either way.
//
// A refusal is NOT a schema mismatch: retrying the same prompt reproduces it.
// Genkit used to surface it as an output-validation failure, which made
// verify-retry loops burn attempts on a model that had already said no.
var ErrModelBlocked = ai.ErrGenerationBlocked

// callerStatuses are the classified failures whose cause is the REQUEST, not
// the service: the provider will reject a retry identically, and their
// appearance says nothing about provider health.
//
// FAILED_PRECONDITION is included because that is the status family
// ErrModelBlocked belongs to (a refusal, not an outage).
var callerStatuses = map[status.Name]bool{
	status.InvalidArgument:    true, // 400 — bad request / bad config
	status.NotFound:           true, // 404 — unknown model or endpoint
	status.PermissionDenied:   true, // 403
	status.Unauthenticated:    true, // 401 — bad or missing key
	status.AlreadyExists:      true, // 409
	status.FailedPrecondition: true, // refused / interrupted before the turn
	status.OutOfRange:         true,
	status.Unimplemented:      true, // 501 — this provider cannot do that
}

// IsCallerFailure reports whether err was caused by the request rather than by
// an unhealthy provider — a bad prompt, an unknown model, a rejected
// credential, or a safety refusal.
//
// Use it to keep retry and circuit-breaking honest: an uncapped "count every
// error" policy lets a caller-side mistake (which retries cannot fix) open a
// circuit breaker and mask a real outage.
//
// Deliberately NOT caller-side: DEADLINE_EXCEEDED and CANCELLED count as
// infrastructure failures. A caller-imposed short timeout would arguably be the
// caller's fault, but in practice a deadline at a model boundary means the
// provider was slow, and treating it as healthy would keep hammering it.
//
// Unclassified errors report false: the safe default is to treat them as
// infrastructure failures and keep counting them.
func IsCallerFailure(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrModelBlocked) {
		return true
	}
	name, ok := status.Classified(err)
	if !ok {
		return false
	}
	return callerStatuses[name]
}

// IsInfraFailure reports whether err should count against provider health
// (outage, throttling, timeout, internal error, or an unknown failure).
// It is the negation of [IsCallerFailure] and is the default predicate a
// circuit breaker should use.
func IsInfraFailure(err error) bool {
	return err != nil && !IsCallerFailure(err)
}
