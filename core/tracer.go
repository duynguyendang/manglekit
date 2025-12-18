package core

import "context"

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

// SetAttributes is a no-op.
func (n *NopSpan) SetAttributes(attributes map[string]any) {}

// SetStatus is a no-op.
func (n *NopSpan) SetStatus(code string, msg string) {}

// RecordError is a no-op.
func (n *NopSpan) RecordError(err error) {}
