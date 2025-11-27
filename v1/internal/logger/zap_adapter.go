package logger

import (
	"github.com/duynguyendang/manglekit/v1/core"
	"go.uber.org/zap"
)

// ZapAdapter wraps a zap.SugaredLogger so it satisfies the core.Logger interface.
type ZapAdapter struct {
	sugared *zap.SugaredLogger
}

// NewZapAdapter constructs a ZapAdapter from the provided sugared logger.
func NewZapAdapter(s *zap.SugaredLogger) *ZapAdapter {
	if s == nil {
		return nil
	}
	return &ZapAdapter{sugared: s}
}

func (z *ZapAdapter) Debugf(msg string, kv ...any) {
	if z == nil || z.sugared == nil {
		return
	}
	z.sugared.Debugw(msg, kv...)
}

func (z *ZapAdapter) Infof(msg string, kv ...any) {
	if z == nil || z.sugared == nil {
		return
	}
	z.sugared.Infow(msg, kv...)
}

func (z *ZapAdapter) Warnf(msg string, kv ...any) {
	if z == nil || z.sugared == nil {
		return
	}
	z.sugared.Warnw(msg, kv...)
}

func (z *ZapAdapter) Errorf(msg string, kv ...any) {
	if z == nil || z.sugared == nil {
		return
	}
	z.sugared.Errorw(msg, kv...)
}

func (z *ZapAdapter) With(kv ...any) core.Logger {
	if z == nil || z.sugared == nil {
		return z
	}
	return &ZapAdapter{sugared: z.sugared.With(kv...)}
}

var _ core.Logger = (*ZapAdapter)(nil)
