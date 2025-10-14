package logger

import (
	"fmt"
	"strings"

	"github.com/duynguyendang/manglekit/core"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Config captures the minimal knobs exposed for the default zap-backed logger.
type Config struct {
	Level  string
	Format string
}

// New constructs a zap-backed core.Logger using the provided configuration.
// When Format is "json" the production encoder is used, otherwise the
// development encoder is selected for human-friendly console output.
func New(cfg Config) (core.Logger, error) {
	format := strings.ToLower(strings.TrimSpace(cfg.Format))
	if format == "" {
		format = "json"
	}

	var zapCfg zap.Config
	switch format {
	case "json":
		zapCfg = zap.NewProductionConfig()
	case "console":
		zapCfg = zap.NewDevelopmentConfig()
	default:
		return nil, fmt.Errorf("unsupported log format: %s", cfg.Format)
	}

	level := strings.ToLower(strings.TrimSpace(cfg.Level))
	if level == "" {
		level = "info"
	}
	var zapLevel zapcore.Level
	switch level {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn", "warning":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	default:
		return nil, fmt.Errorf("unsupported log level: %s", cfg.Level)
	}
	zapCfg.Level = zap.NewAtomicLevelAt(zapLevel)

	zapLogger, err := zapCfg.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build zap logger: %w", err)
	}

	return NewZapAdapter(zapLogger.Sugar()), nil
}
