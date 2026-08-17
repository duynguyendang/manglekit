// Package exitcode defines the mkit CLI exit-code contract and maps errors
// to exit codes. Keep in sync with the root command help text in cmd/mkit/main.go.
package exitcode

import (
	"errors"
	"fmt"

	"github.com/duynguyendang/manglekit/core"
)

// mkit exit codes. Documented in the root command help.
const (
	// Success means the command completed without error.
	Success = 0
	// PolicyDeny means a governance policy blocked the requested action.
	PolicyDeny = 1
	// Usage means the command was invoked with invalid flags or arguments.
	Usage = 2
	// Runtime means the command failed for an operational reason
	// (I/O, engine, provider failures).
	Runtime = 3
)

// UsageError marks errors caused by invalid CLI usage (bad flags or
// arguments). The CLI exits with code Usage for these.
type UsageError struct {
	Err error
}

func (e *UsageError) Error() string { return e.Err.Error() }
func (e *UsageError) Unwrap() error { return e.Err }

// UsageErrorf returns an error that maps to the Usage exit code.
func UsageErrorf(format string, args ...any) error {
	return &UsageError{Err: fmt.Errorf(format, args...)}
}

// IsUsageError reports whether err was caused by invalid CLI usage.
func IsUsageError(err error) bool {
	var ue *UsageError
	return errors.As(err, &ue)
}

// CodeFor maps an error to the mkit exit code.
func CodeFor(err error) int {
	switch {
	case err == nil:
		return Success
	case errors.Is(err, core.ErrPolicyViolation), errors.Is(err, core.ErrAlignment):
		return PolicyDeny
	case IsUsageError(err):
		return Usage
	default:
		return Runtime
	}
}
