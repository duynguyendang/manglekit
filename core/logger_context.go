package core

import "context"

// loggerContextKey is an unexported type used as the key for storing
// the Logger in a context.Context. Using a dedicated type prevents
// collisions with keys from other packages.
type loggerContextKey int

// loggerKey is the single instance of loggerContextKey used as the key.
const loggerKey loggerContextKey = 0

// defaultLogger is a NopLogger instance used when no logger is found in the context.
var defaultLogger Logger = NopLogger{}

// LoggerWithContext returns a new context that carries the provided Logger.
// Downstream code can retrieve it with LoggerFromContext.
//
// Usage:
//
//	ctx = core.LoggerWithContext(ctx, myLogger)
func LoggerWithContext(ctx context.Context, logger Logger) context.Context {
	if logger == nil {
		logger = defaultLogger
	}
	return context.WithValue(ctx, loggerKey, logger)
}

// LoggerFromContext retrieves the Logger from the context. If no Logger
// was stored, it returns a NopLogger that discards all log messages,
// ensuring callers never receive nil.
//
// Usage:
//
//	logger := core.LoggerFromContext(ctx)
//	logger.Info("processing request", "request_id", reqID)
func LoggerFromContext(ctx context.Context) Logger {
	if logger, ok := ctx.Value(loggerKey).(Logger); ok && logger != nil {
		return logger
	}
	return defaultLogger
}

// SetDefaultLogger sets a global default logger that is returned by
// LoggerFromContext when no logger is present in the context.
// This should be called early in application initialization if a
// non-NopLogger default is desired.
func SetDefaultLogger(logger Logger) {
	if logger == nil {
		logger = NopLogger{}
	}
	defaultLogger = logger
}
