package telemetry

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric/noop"
	otelnoop "go.opentelemetry.io/otel/trace/noop"
)

func TestMeterProvider_DisabledByDefault(t *testing.T) {
	mp := NewMeterProvider()
	if mp.Enabled() {
		t.Error("expected MeterProvider to be disabled with no meter set")
	}
	if m := mp.Meter("x"); m != nil {
		t.Errorf("expected nil meter when disabled, got %v", m)
	}
}

func TestMeterProvider_SetMeterEnables(t *testing.T) {
	mp := NewMeterProvider()
	mp.SetMeter(noop.NewMeterProvider().Meter("test"))

	if !mp.Enabled() {
		t.Error("expected MeterProvider to be enabled after SetMeter")
	}
	if m := mp.Meter("any"); m == nil {
		t.Error("expected non-nil meter after SetMeter")
	}
}

func TestMeterProvider_RegisterMeter(t *testing.T) {
	mp := NewMeterProvider()
	registered := noop.NewMeterProvider().Meter("named")
	mp.RegisterMeter("named", registered)

	if mp.Enabled() {
		t.Error("RegisterMeter should not enable the default meter")
	}
	if m := mp.Meter("named"); m == nil {
		t.Error("expected registered meter to be retrievable by name")
	}
	if m := mp.Meter("missing"); m != nil {
		t.Errorf("expected nil for unregistered name, got %v", m)
	}
}

func TestTagsToAttrs(t *testing.T) {
	attrs := TagsToAttrs(map[string]string{"env": "prod", "region": "us"})
	if len(attrs) != 2 {
		t.Fatalf("expected 2 attributes, got %d", len(attrs))
	}
	want := map[attribute.Key]string{
		"env":    "prod",
		"region": "us",
	}
	for _, a := range attrs {
		if want[a.Key] != a.Value.AsString() {
			t.Errorf("unexpected attribute %s=%s", a.Key, a.Value.AsString())
		}
	}
}

func TestOTelTracer_NilReturnsNop(t *testing.T) {
	tr := NewOTelTracer(nil)
	ctx, span := tr.Start(context.Background(), "op")
	if ctx == nil {
		t.Error("expected non-nil context")
	}
	span.RecordError(context.Canceled)
	span.End()
}

func TestOTelTracer_WrapsRealTracer(t *testing.T) {
	tr := NewOTelTracer(otelnoop.NewTracerProvider().Tracer("test"))
	ctx, span := tr.Start(context.Background(), "op")
	if ctx == nil {
		t.Error("expected non-nil context")
	}
	span.RecordError(context.Canceled)
	span.SetStatus("error", "boom")
	span.End()
}
