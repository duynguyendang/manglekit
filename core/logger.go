package core

// Logger defines a vendor-neutral interface for structured logging. The
// methods follow a "message + key/value" calling convention so callers can
// attach context without committing to any specific backend semantics.
//
// All Manglekit components and user applications should use this interface
// for logging to ensure consistent, structured log output across the system.
type Logger interface {
	// Debug records verbose diagnostic information about control flow or
	// intermediate state. Arguments are interpreted as key/value pairs.
	Debug(msg string, fields ...any)

	// Info records high-level lifecycle events such as component start or
	// stop. Arguments are interpreted as key/value pairs.
	Info(msg string, fields ...any)

	// Warn records recoverable issues that deserve operator attention but
	// do not stop execution. Arguments are interpreted as key/value pairs.
	Warn(msg string, fields ...any)

	// Error records failures. Implementations should treat the "error"
	// key specially when present. Arguments are interpreted as key/value pairs.
	Error(msg string, fields ...any)

	// With returns a child logger that automatically appends the supplied
	// key/value pairs to every log record.
	With(fields ...any) Logger
}

// NopLogger is a Logger implementation that discards all log messages.
// It is useful as a default when no logger is configured.
type NopLogger struct{}

// Debug implements Logger.Debug as a no-op.
func (NopLogger) Debug(msg string, fields ...any) {}

// Info implements Logger.Info as a no-op.
func (NopLogger) Info(msg string, fields ...any) {}

// Warn implements Logger.Warn as a no-op.
func (NopLogger) Warn(msg string, fields ...any) {}

// Error implements Logger.Error as a no-op.
func (NopLogger) Error(msg string, fields ...any) {}

// With implements Logger.With, returning the same NopLogger.
func (n NopLogger) With(fields ...any) Logger { return n }

// Ensure NopLogger satisfies the Logger interface at compile time.
var _ Logger = NopLogger{}
