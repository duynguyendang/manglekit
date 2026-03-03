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
	// ErrInputValidation is returned when input conversion or parsing fails.
	ErrInputValidation = errors.New("input validation error")
	// ErrPolicyViolation is returned when a supervisor blocks due to policy violation.
	ErrPolicyViolation = errors.New("policy violation")
	// ErrSupervisorFailure is returned when the supervisor itself fails.
	ErrSupervisorFailure = errors.New("supervisor failure")
)

// AlignmentError is a structured error that carries a specific intervention message.
// It wraps ErrAlignment to ensure standard error matching works.
type AlignmentError struct {
	Message string
	RuleID  string
}

func (e *AlignmentError) Error() string {
	if e.RuleID != "" {
		return fmt.Sprintf("[INTERVENTION] [%s]: %s", e.RuleID, e.Message)
	}
	return fmt.Sprintf("[INTERVENTION]: %s", e.Message)
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

// InputError is a structured error that indicates a failure in input validation or fact conversion.
// It wraps ErrInputValidation to ensure distinction from system errors.
type InputError struct {
	Err error
}

func (e *InputError) Error() string {
	return fmt.Sprintf("input validation error: %v", e.Err)
}

func (e *InputError) Is(target error) bool {
	return target == ErrInputValidation
}

func (e *InputError) Unwrap() error {
	return ErrInputValidation
}

// IsInputError checks if the error is an InputError.
func IsInputError(err error) bool {
	return errors.Is(err, ErrInputValidation)
}

// NewAlignmentError creates a new AlignmentError with the given message and rule ID.
func NewAlignmentError(message, ruleID string) *AlignmentError {
	return &AlignmentError{
		Message: message,
		RuleID:  ruleID,
	}
}

// WrapInputError wraps an error as an InputError.
func WrapInputError(err error) *InputError {
	return &InputError{Err: err}
}

// PolicyViolationError indicates the supervisor blocked execution due to a policy violation.
type PolicyViolationError struct {
	Tier       string
	RuleID     string
	Violation  string
	Suggestion string
}

func (e *PolicyViolationError) Error() string {
	return fmt.Sprintf("policy violation: blocked at tier %s (rule: %s)", e.Tier, e.RuleID)
}

func (e *PolicyViolationError) Is(target error) bool {
	return target == ErrPolicyViolation
}

func (e *PolicyViolationError) Unwrap() error {
	return ErrPolicyViolation
}

// IsPolicyViolationError checks if the error is a PolicyViolationError.
func IsPolicyViolationError(err error) bool {
	return errors.Is(err, ErrPolicyViolation)
}

// NewPolicyViolationError creates a new PolicyViolationError.
func NewPolicyViolationError(tier, ruleID, violation, suggestion string) *PolicyViolationError {
	return &PolicyViolationError{
		Tier:       tier,
		RuleID:     ruleID,
		Violation:  violation,
		Suggestion: suggestion,
	}
}

// SupervisorError indicates a failure in the supervisor itself.
type SupervisorError struct {
	Reason error
}

func (e *SupervisorError) Error() string {
	return fmt.Sprintf("supervisor failure: %v", e.Reason)
}

func (e *SupervisorError) Is(target error) bool {
	return target == ErrSupervisorFailure
}

func (e *SupervisorError) Unwrap() error {
	return ErrSupervisorFailure
}

// IsSupervisorError checks if the error is a SupervisorError.
func IsSupervisorError(err error) bool {
	return errors.Is(err, ErrSupervisorFailure)
}
