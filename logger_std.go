package manglekit

import (
	"log/slog"
	"os"
	"sync"

	"github.com/duynguyendang/manglekit/core"
)

var (
	defaultLogger core.Logger
	once          sync.Once
)

// getDefaultLogger returns the singleton default logger.
// It initializes it with a slog text handler on standard output if not already initialized.
func getDefaultLogger() core.Logger {
	once.Do(func() {
		// Use TextHandler for readability, default to Info level
		h := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
		defaultLogger = &slogAdapter{l: slog.New(h)}
	})
	return defaultLogger
}

// NewStdLogger allows manual usage of the default logger
func NewStdLogger() core.Logger {
	return getDefaultLogger()
}

// slogAdapter implements core.Logger using log/slog.
type slogAdapter struct {
	l *slog.Logger
}

// Debug records verbose diagnostic information.
func (s *slogAdapter) Debug(msg string, args ...any) {
	s.l.Debug(msg, args...)
}

// Info records high-level lifecycle events.
func (s *slogAdapter) Info(msg string, args ...any) {
	s.l.Info(msg, args...)
}

// Warn records recoverable issues.
func (s *slogAdapter) Warn(msg string, args ...any) {
	s.l.Warn(msg, args...)
}

// Error records failures.
func (s *slogAdapter) Error(msg string, args ...any) {
	s.l.Error(msg, args...)
}

// With returns a child logger with the appended key/value pairs.
func (s *slogAdapter) With(args ...any) core.Logger {
	return &slogAdapter{l: s.l.With(args...)}
}

// Ensure slogAdapter satisfies the Logger interface at compile time.
var _ core.Logger = &slogAdapter{}
