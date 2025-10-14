package logger

import (
	"fmt"
	"strings"

	"github.com/duynguyendang/manglekit/core"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Options controls how the zap-backed logger is constructed.
type Options struct {
	Level  string
	Format string
}

// New creates a zap-based logger that satisfies the core.Logger interface.
// It accepts Options to control the logging level and output encoding.
func New(opts Options) (core.Logger, error) {
	cfg, err := buildConfig(opts)
	if err != nil {
		return nil, err
	}
	zl, err := cfg.Build()
	if err != nil {
		return nil, fmt.Errorf("logger: failed to build zap logger: %w", err)
	}
	return &ZapAdapter{logger: zl.Sugar()}, nil
}

func buildConfig(opts Options) (zap.Config, error) {
	format := strings.ToLower(strings.TrimSpace(opts.Format))
	level := strings.ToLower(strings.TrimSpace(opts.Level))

	var cfg zap.Config
	switch format {
	case "", "console":
		cfg = zap.NewDevelopmentConfig()
	case "json":
		cfg = zap.NewProductionConfig()
	default:
		return zap.Config{}, fmt.Errorf("logger: unsupported format %q", opts.Format)
	}

	if level == "" {
		level = "info"
	}
	zapLevel, err := parseLevel(level)
	if err != nil {
		return zap.Config{}, err
	}
	cfg.Level = zap.NewAtomicLevelAt(zapLevel)

	if format == "console" || format == "" {
		cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	return cfg, nil
}

func parseLevel(level string) (zapcore.Level, error) {
	switch level {
	case "debug":
		return zap.DebugLevel, nil
	case "info":
		return zap.InfoLevel, nil
	case "warn", "warning":
		return zap.WarnLevel, nil
	case "error":
		return zap.ErrorLevel, nil
	default:
		return zapcore.Level(0), fmt.Errorf("logger: unsupported level %q", level)
	}
}
