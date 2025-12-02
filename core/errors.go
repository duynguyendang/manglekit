package core

import (
	"errors"
	"fmt"
)

var (
	// ErrPolicyViolation is returned when a policy blocks an action.
	ErrPolicyViolation = errors.New("policy violation")
	// ErrSystemError is returned when an unexpected error occurs.
	ErrSystemError = errors.New("system error")
)

// PolicyViolationError is a structured error that carries a specific violation message.
// It wraps ErrPolicyViolation to ensure standard error matching works.
type PolicyViolationError struct {
	Message string
}

func (e *PolicyViolationError) Error() string {
	return fmt.Sprintf("policy violation: %s", e.Message)
}

func (e *PolicyViolationError) Is(target error) bool {
	return target == ErrPolicyViolation
}
