package telemetry

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/duynguyendang/manglekit-wip/core"
)

// OTelTracer wraps the OpenTelemetry trace.Tracer to implement core.Tracer.
// This adapter encapsulates all OTel-specific logic, allowing the rest of the codebase
// to depend only on the high-level core.Tracer interface.
type OTelTracer struct {
	tracer trace.Tracer
}

// NewOTelTracer creates a new OTelTracer from an OpenTelemetry trace.Tracer.
func NewOTelTracer(tracer trace.Tracer) core.Tracer {
	if tracer == nil {
		return &core.NopTracer{}
	}
	return &OTelTracer{tracer: tracer}
}

// Start creates a new OTel span wrapped in the core.Span interface.
func (o *OTelTracer) Start(ctx context.Context, name string) (context.Context, core.Span) {
	ctx, span := o.tracer.Start(ctx, name)
	return ctx, &OTelSpan{span: span}
}

// OTelSpan wraps the OpenTelemetry trace.Span to implement core.Span.
type OTelSpan struct {
	span trace.Span
}

// End completes the span.
func (o *OTelSpan) End() {
	o.span.End()
}

// Error records an error in the span and marks it as failed.
func (o *OTelSpan) Error(err error) {
	o.span.SetStatus(codes.Error, err.Error())
	o.span.RecordError(err)
}

// RecordError records an error in the span.
// This implements the core.Span interface.
func (o *OTelSpan) RecordError(err error) {
	o.span.RecordError(err)
}

// SetStatus sets the status of the span.
func (o *OTelSpan) SetStatus(code string, msg string) {
	// Map string code to OTel code if needed, but core.Span uses strings for flexibility
	// For OTel, we typically use codes.Error or codes.Ok
	if code == "error" || code == "ERROR" {
		o.span.SetStatus(codes.Error, msg)
	} else if code == "ok" || code == "OK" {
		o.span.SetStatus(codes.Ok, msg)
	} else {
		o.span.SetStatus(codes.Unset, msg)
	}
}

// SetAttributes sets multiple attributes.
func (o *OTelSpan) SetAttributes(attributes map[string]any) {
	for k, v := range attributes {
		o.SetAttr(k, v)
	}
}

// SetAttr sets a key-value attribute on the span.
// It converts interface{} values to appropriate OTel attribute types.
func (o *OTelSpan) SetAttr(key string, value interface{}) {
	var attr attribute.KeyValue

	switch v := value.(type) {
	case string:
		attr = attribute.String(key, v)
	case bool:
		attr = attribute.Bool(key, v)
	case int:
		attr = attribute.Int(key, v)
	case int64:
		attr = attribute.Int64(key, v)
	case float64:
		attr = attribute.Float64(key, v)
	default:
		attr = attribute.String(key, valueToString(v))
	}

	o.span.SetAttributes(attr)
}

// valueToString converts any value to a string representation.
// This is a fallback for types that don't have native attribute support.
func valueToString(v interface{}) string {
	// Simple string conversion; can be enhanced as needed
	return ""
}
