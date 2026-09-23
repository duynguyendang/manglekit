package ai

import (
	"errors"
	"fmt"
	"testing"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core/status"
)

func TestIsCallerFailure(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"unclassified", errors.New("socket hung up"), false},
		{"400 bad request", status.Errorf(status.ErrInvalidArgument, "temperature out of range"), true},
		{"404 unknown model", status.Errorf(status.ErrNotFound, "model %q not found", "gpt-9"), true},
		{"401 bad key", status.Errorf(status.ErrUnauthenticated, "invalid api key"), true},
		{"403", status.Errorf(status.ErrPermissionDenied, "no access"), true},
		{"501 unsupported", status.Errorf(status.ErrUnimplemented, "no tools"), true},
		// Subtypes resolve to their base status.
		{"invalid input (subtype of 400)", status.Errorf(status.ErrInvalidInput, "bad tool args"), true},
		{"action not found (subtype of 404)", status.Errorf(status.ErrActionNotFound, "no such action"), true},
		// Provider health, not the request: these MUST count as failures.
		{"503 unavailable", status.Errorf(status.ErrUnavailable, "backend busy"), false},
		{"429 throttled", status.Errorf(status.ErrResourceExhausted, "quota"), false},
		{"504 deadline", status.Errorf(status.ErrDeadlineExceeded, "too slow"), false},
		{"500 internal", status.Errorf(status.ErrInternal, "boom"), false},
		// Upstream classifies a schema-invalid OUTPUT as INTERNAL — the fault
		// is on the producing side and a retry may well succeed. Pinned here
		// so the mapping cannot drift silently.
		{"invalid output", status.Errorf(status.ErrInvalidOutput, "model output mismatched schema"), false},
		// A refusal is the caller's problem in the practical sense: retrying
		// the same prompt cannot help.
		{"safety refusal", status.Errorf(ai.ErrGenerationBlocked, "refused"), true},
		// Wrapping must not erase the classification.
		{"wrapped 400", fmt.Errorf("generate: %w", status.Errorf(status.ErrInvalidArgument, "x")), true},
		{"double wrapped 503", fmt.Errorf("a: %w", fmt.Errorf("b: %w", status.Errorf(status.ErrUnavailable, "x"))), false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsCallerFailure(tc.err); got != tc.want {
				t.Errorf("IsCallerFailure(%v) = %v, want %v", tc.err, got, tc.want)
			}
			if got, want := IsInfraFailure(tc.err), !tc.want && tc.err != nil; got != want {
				t.Errorf("IsInfraFailure(%v) = %v, want %v", tc.err, got, want)
			}
		})
	}
}

// TestErrModelBlockedIsGenkitSentinel keeps the re-export honest: callers may
// branch on either value and errors.Is must agree.
func TestErrModelBlockedIsGenkitSentinel(t *testing.T) {
	err := status.Errorf(ai.ErrGenerationBlocked, "policy refused")
	if !errors.Is(err, ErrModelBlocked) {
		t.Fatal("ErrModelBlocked must match genkit's ai.ErrGenerationBlocked")
	}
	if !errors.Is(err, ai.ErrGenerationBlocked) {
		t.Fatal("re-export must not break the upstream sentinel")
	}
}

// TestBlockedIsNotSchemaMismatch is the A1 regression: a refusal and a
// malformed output are different failures and a retry loop must be able to
// tell them apart.
func TestBlockedIsNotSchemaMismatch(t *testing.T) {
	blocked := status.Errorf(ai.ErrGenerationBlocked, "refused")
	schema := status.Errorf(status.ErrInvalidOutput, "does not match schema")

	if errors.Is(schema, ErrModelBlocked) {
		t.Fatal("a schema mismatch must not look like a refusal")
	}
	if errors.Is(blocked, status.ErrInvalidOutput) {
		t.Fatal("a refusal must not look like a schema mismatch")
	}
}
