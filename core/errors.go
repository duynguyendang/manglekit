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
}

func (e *AlignmentError) Error() string {
	return fmt.Sprintf("[INTERVENTION]: %s", e.Message)
}

func (e *AlignmentError) Is(target error) bool {
	return target == ErrAlignment
}
