package logger

import (
	"strings"

	"github.com/duynguyendang/manglekit/core"
	"go.uber.org/zap"
)

type ZapAdapter struct {
	logger *zap.SugaredLogger
}

func (z *ZapAdapter) Debugf(template string, args ...any) {
	logSugared(z.logger.Debugf, z.logger.Debugw, template, args...)
}

func (z *ZapAdapter) Infof(template string, args ...any) {
	logSugared(z.logger.Infof, z.logger.Infow, template, args...)
}

func (z *ZapAdapter) Warnf(template string, args ...any) {
	logSugared(z.logger.Warnf, z.logger.Warnw, template, args...)
}

func (z *ZapAdapter) Errorf(template string, args ...any) {
	logSugared(z.logger.Errorf, z.logger.Errorw, template, args...)
}

func (z *ZapAdapter) With(args ...any) core.Logger {
	return &ZapAdapter{logger: z.logger.With(args...)}
}

func logSugared(formatFn func(string, ...any), structuredFn func(string, ...any), template string, args ...any) {
	if len(args) == 0 {
		formatFn(template)
		return
	}
	if !strings.Contains(template, "%") && len(args)%2 == 0 {
		structuredFn(template, args...)
		return
	}
	formatFn(template, args...)
}
