package sdk

import (
	"context"

	"github.com/duynguyendang/manglekit/internal/telemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

// WithStdoutTracer configures the client to use a standard output tracer.
// This is useful for development and debugging to see traces in the console.
func WithStdoutTracer() ClientOption {
	return func(c *Client) error {
		exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
		if err != nil {
			return err
		}

		tp := sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(exporter),
			sdktrace.WithResource(resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceName("manglekit-app"),
			)),
		)

		otel.SetTracerProvider(tp)

		otelTracer := tp.Tracer(TracerName)
		c.otelTracer = otelTracer
		c.tracer = telemetry.NewOTelTracer(otelTracer)

		c.shutdownFunc = func(ctx context.Context) error {
			return tp.Shutdown(ctx)
		}
		return nil
	}
}
