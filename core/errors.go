package core

import "errors"

var (
	// ErrPolicyViolation is returned when a policy blocks an action.
	ErrPolicyViolation = errors.New("policy violation")
	// ErrSystemError is returned when an unexpected error occurs.
	ErrSystemError = errors.New("system error")
)
