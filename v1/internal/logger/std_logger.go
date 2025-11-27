package logger

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/duynguyendang/manglekit/v1/core"
)

// StdLogger is a lightweight fallback that writes structured records to stdout.
type StdLogger struct {
	base   *log.Logger
	fields []any
	mu     sync.Mutex
}

// NewStdLogger constructs a StdLogger backed by the standard library logger.
func NewStdLogger() core.Logger {
	return &StdLogger{
		base: log.New(os.Stdout, "", log.LstdFlags|log.LUTC),
	}
}

func (l *StdLogger) With(kv ...any) core.Logger {
	child := &StdLogger{base: l.base}
	child.fields = append(child.fields, l.fields...)
	child.fields = append(child.fields, kv...)
	return child
}

func (l *StdLogger) Debugf(msg string, kv ...any) {
	l.log("DEBUG", msg, kv...)
}

func (l *StdLogger) Infof(msg string, kv ...any) {
	l.log("INFO", msg, kv...)
}

func (l *StdLogger) Warnf(msg string, kv ...any) {
	l.log("WARN", msg, kv...)
}

func (l *StdLogger) Errorf(msg string, kv ...any) {
	l.log("ERROR", msg, kv...)
}

func (l *StdLogger) log(level, msg string, kv ...any) {
	if l == nil || l.base == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	builder := strings.Builder{}
	builder.WriteString("[")
	builder.WriteString(level)
	builder.WriteString("] ")
	builder.WriteString(msg)

	args := append([]any{}, l.fields...)
	args = append(args, kv...)
	if len(args) > 0 {
		builder.WriteString(" ")
		builder.WriteString(formatKeyValues(args))
	}

	l.base.Println(builder.String())
}

func formatKeyValues(kv []any) string {
	if len(kv) == 0 {
		return ""
	}
	pairs := make([]string, 0, len(kv)/2+1)
	for i := 0; i < len(kv); i += 2 {
		key := toString(kv[i])
		var value any
		if i+1 < len(kv) {
			value = kv[i+1]
		}
		pairs = append(pairs, key+"="+toString(value))
	}
	return strings.Join(pairs, " ")
}

func toString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	default:
		return strings.TrimSpace(strings.ReplaceAll(fmt.Sprintf("%v", v), "\n", " "))
	}
}

var _ core.Logger = (*StdLogger)(nil)
