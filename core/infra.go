package core

import "context"

// Breaker implements the Circuit Breaker pattern.
type Breaker interface {
	Execute(req func() (any, error)) (any, error)
	Name() string
}

// Meter abstracts metrics (Counters, Gauges).
type Meter interface {
	Counter(name string, val int64, tags map[string]string)
	Histogram(name string, val float64, tags map[string]string)
}

// Tracer abstracts distributed tracing.
type Tracer interface {
	// Start creates a new span with the given name and returns a context and span.
	Start(ctx context.Context, name string) (context.Context, Span)
}

// Span represents an active operation being traced.
type Span interface {
	End()
	SetAttributes(attributes map[string]any)
	SetStatus(code string, msg string)
	RecordError(err error)
}
