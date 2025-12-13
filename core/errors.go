package core

import (
	"errors"
	"fmt"
)

var (
	// ErrAlignment is returned when a blueprint alignment check blocks an action.
	ErrAlignment = errors.New("alignment error")
	// ErrSystemError is returned when an unexpected error occurs.
	ErrSystemError = errors.New("system error")
)

// AlignmentError is a structured error that carries a specific intervention message.
// It wraps ErrAlignment to ensure standard error matching works.
type AlignmentError struct {
	Message string
	RuleID  string
}

func (e *AlignmentError) Error() string {
	if e.RuleID != "" {
		return fmt.Sprintf("[ALIGNMENT INTERVENTION] [%s]: %s", e.RuleID, e.Message)
	}
	return fmt.Sprintf("[ALIGNMENT INTERVENTION]: %s", e.Message)
}

func (e *AlignmentError) Is(target error) bool {
	return target == ErrAlignment
}

func (e *AlignmentError) Unwrap() error {
	return ErrAlignment
}

// IsAlignmentError checks if the error is an AlignmentError.
func IsAlignmentError(err error) bool {
	return errors.Is(err, ErrAlignment)
}
