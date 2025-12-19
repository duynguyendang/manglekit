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
	return e.Err
}

// IsInputError checks if the error is an InputError.
func IsInputError(err error) bool {
	return errors.Is(err, ErrInputValidation)
}
