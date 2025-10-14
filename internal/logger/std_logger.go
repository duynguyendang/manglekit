package logger

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/duynguyendang/manglekit/core"
)

type stdLogger struct {
	base   *log.Logger
	fields []any
}

// NewStdLogger returns a minimal core.Logger implementation backed by the
// standard library logger. It is used as a guaranteed fallback when no other
// logger is configured.
func NewStdLogger() core.Logger {
	return &stdLogger{base: log.New(os.Stdout, "", 0)}
}

func (l *stdLogger) Debugf(template string, args ...any) {
	l.log("DEBUG", template, args...)
}

func (l *stdLogger) Infof(template string, args ...any) {
	l.log("INFO", template, args...)
}

func (l *stdLogger) Warnf(template string, args ...any) {
	l.log("WARN", template, args...)
}

func (l *stdLogger) Errorf(template string, args ...any) {
	l.log("ERROR", template, args...)
}

func (l *stdLogger) With(args ...any) core.Logger {
	merged := append([]any{}, l.fields...)
	merged = append(merged, args...)
	return &stdLogger{base: l.base, fields: merged}
}

func (l *stdLogger) log(level, template string, args ...any) {
	message, kv := splitMessage(template, args...)
	fields := append([]any{}, l.fields...)
	if len(kv) > 0 {
		fields = append(fields, kv...)
	}
	line := buildLine(level, message, fields)
	l.base.Println(line)
}

func splitMessage(template string, args ...any) (string, []any) {
	if len(args) == 0 {
		return template, nil
	}
	if strings.Contains(template, "%") {
		return fmt.Sprintf(template, args...), nil
	}
	if len(args)%2 == 0 {
		return template, args
	}
	return fmt.Sprintf("%s %v", template, args), nil
}

func buildLine(level, message string, fields []any) string {
	var b strings.Builder
	b.WriteString("[")
	b.WriteString(level)
	b.WriteString("] ")
	b.WriteString(message)
	if len(fields) == 0 {
		return b.String()
	}
	b.WriteRune(' ')
	b.WriteString(formatFields(fields))
	return b.String()
}

func formatFields(fields []any) string {
	var b strings.Builder
	for i := 0; i < len(fields); i += 2 {
		if i > 0 {
			b.WriteRune(' ')
		}
		key := fmt.Sprint(fields[i])
		var value any = "<missing>"
		if i+1 < len(fields) {
			value = fields[i+1]
		}
		b.WriteString(key)
		b.WriteRune('=')
		b.WriteString(fmt.Sprint(value))
	}
	return b.String()
}
