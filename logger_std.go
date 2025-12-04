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

// getDefaultLogger returns the Singleton Slog instance (Text Handler)
func getDefaultLogger() core.Logger {
	once.Do(func() {
		h := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
		defaultLogger = &slogAdapter{l: slog.New(h)}
	})
	return defaultLogger
}

// NewStdLogger exposes the default logger for manual use
func NewStdLogger() core.Logger {
	return getDefaultLogger()
}

// slogAdapter adapts slog to core.Logger
type slogAdapter struct {
	l *slog.Logger
}

func (s *slogAdapter) Debug(msg string, args ...any) { s.l.Debug(msg, args...) }
func (s *slogAdapter) Info(msg string, args ...any)  { s.l.Info(msg, args...) }
func (s *slogAdapter) Warn(msg string, args ...any)  { s.l.Warn(msg, args...) }
func (s *slogAdapter) Error(msg string, args ...any) { s.l.Error(msg, args...) }

func (s *slogAdapter) With(args ...any) core.Logger {
	return &slogAdapter{l: s.l.With(args...)}
}
