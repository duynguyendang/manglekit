package core

import (
	"errors"
	"fmt"
	"strings"
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
//
// The ActionName, MatchedRule, and Tier fields carry provenance from the
// engine's AuditTrail so callers can report which action was denied, by
// which rule, at which governance tier. They are optional; older callers
// that only set Message/RuleID keep working.
type AlignmentError struct {
	Message string
	RuleID  string
	// ActionName is the supervised action the gate was evaluating, if known.
	ActionName string
	// MatchedRule is the text of the rule/fact that matched the denial
	// (e.g. `halt("Req", "...", "T1")`).
	MatchedRule string
	// Tier is the governance tier of the matched rule (core.Tier*).
	Tier Tier
}

func (e *AlignmentError) Error() string {
	var b strings.Builder
	b.WriteString("[INTERVENTION]")
	if e.RuleID != "" {
		b.WriteString(" [" + e.RuleID + "]")
	}
	if e.ActionName != "" {
		b.WriteString(" (action: " + e.ActionName)
		if e.Tier != "" {
			b.WriteString(", tier: " + string(e.Tier))
		}
		b.WriteString(")")
	}
	b.WriteString(": ")
	b.WriteString(e.Message)
	return b.String()
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
	// ActionName is the supervised action that was blocked, when known.
	ActionName string
	// MatchedRule is the text of the policy rule/fact that matched the
	// denial, taken from the engine's AuditTrail.
	MatchedRule string
}

func (e *PolicyViolationError) Error() string {
	msg := fmt.Sprintf("policy violation: blocked at tier %s (rule: %s)", e.Tier, e.RuleID)
	if e.ActionName != "" {
		msg += fmt.Sprintf(" (action: %s)", e.ActionName)
	}
	if e.Violation != "" {
		msg += ": " + e.Violation
	}
	if e.MatchedRule != "" {
		msg += fmt.Sprintf(" [matched: %s]", e.MatchedRule)
	}
	if e.Suggestion != "" {
		msg += fmt.Sprintf(" (suggestion: %s)", e.Suggestion)
	}
	return msg
}

func (e *PolicyViolationError) Is(target error) bool {
	return target == ErrPolicyViolation
}

func (e *PolicyViolationError) Unwrap() error {
	return ErrPolicyViolation
}

// IsPolicyViolationError checks if the error represents a policy block,
// regardless of which path produced it. It matches both the supervisor's
// PolicyViolationError (Execute path) and the engine's AlignmentError
// (direct Assess/AssessPlan calls), so block-detection code can use one
// idiom consistently:
//
//	if core.IsPolicyViolationError(err) { /* blocked by policy */ }
//
// Note that AssessPlan reports denies as DecisionHalt in the returned
// Decision (with a nil error); use decision.Outcome == core.DecisionHalt
// there.
func IsPolicyViolationError(err error) bool {
	return errors.Is(err, ErrPolicyViolation) || errors.Is(err, ErrAlignment)
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
