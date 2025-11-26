package core

import "context"

// Tracer defines the capability to start spans for tracing and observability.
// Implementations should handle both OTel and no-op scenarios transparently.
type Tracer interface {
	// Start creates a new span with the given name and returns a context and span.
	// The context should be used for all downstream operations to maintain the trace hierarchy.
	Start(ctx context.Context, name string) (context.Context, Span)
}

// Span represents an active operation being traced.
type Span interface {
	// End completes the span. It should be called with defer to ensure cleanup.
	End()

	// Error records an error in the span and marks the span as failed.
	// This is a convenience method that combines SetAttr("error", "true"), recording the error,
	// and setting the appropriate error status.
	Error(err error)

	// SetAttr sets a key-value attribute on the span.
	// Value can be any type that the underlying tracer supports (typically string, bool, number).
	SetAttr(key string, value interface{})
}

// NopTracer is a no-op implementation of Tracer for when tracing is disabled.
// All methods are safe no-ops that maintain the trace interface contract.
type NopTracer struct{}

// Start returns a no-op span.
func (n *NopTracer) Start(ctx context.Context, name string) (context.Context, Span) {
	return ctx, &NopSpan{}
}

// NopSpan is a no-op implementation of Span for when tracing is disabled.
type NopSpan struct{}

// End is a no-op.
func (n *NopSpan) End() {}

// Error is a no-op.
func (n *NopSpan) Error(err error) {}

// SetAttr is a no-op.
func (n *NopSpan) SetAttr(key string, value interface{}) {}
