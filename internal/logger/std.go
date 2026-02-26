package logger

import (
	"log/slog"
	"os"

	"github.com/duynguyendang/manglekit-wip/core"
)

type SlogAdapter struct {
	logger *slog.Logger
}

func NewDefault() core.Logger {
	return New("info")
}

func New(level string) core.Logger {
	var l slog.Level
	switch level {
	case "debug":
		l = slog.LevelDebug
	case "warn":
		l = slog.LevelWarn
	case "error":
		l = slog.LevelError
	default:
		l = slog.LevelInfo
	}
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: l,
	})
	return &SlogAdapter{
		logger: slog.New(handler),
	}
}

func (s *SlogAdapter) Debug(msg string, args ...any) {
	s.logger.Debug(msg, args...)
}

func (s *SlogAdapter) Info(msg string, args ...any) {
	s.logger.Info(msg, args...)
}

func (s *SlogAdapter) Warn(msg string, args ...any) {
	s.logger.Warn(msg, args...)
}

func (s *SlogAdapter) Error(msg string, args ...any) {
	s.logger.Error(msg, args...)
}

func (s *SlogAdapter) With(args ...any) core.Logger {
	return &SlogAdapter{
		logger: s.logger.With(args...),
	}
}
