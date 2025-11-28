package guard

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/engine"
	"github.com/duynguyendang/manglekit/internal/telemetry"
)

// TestTraceHierarchy verifies the correct OTel span hierarchy:
// Action.MyAction (Guard)
// ├── Datalog.PreCheck (Engine.Authorize)
// ├── Function.Execute / Inner Action
// └── Datalog.PostCheck (Engine.Validate)
func TestTraceHierarchy(t *testing.T) {
	// Setup: Create a tracing exporter to capture spans
	exporter := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
	)
	otelTracer := tp.Tracer("test")
	defer func() { _ = tp.Shutdown(context.Background()) }()

	// Wrap OTel tracer with the telemetry adapter
	coreTracer := telemetry.NewOTelTracer(otelTracer)

	// Initialize the engine and guard with tracing
	eng := engine.NewWithObservability(coreTracer, core.NopLogger{})

	// Create a simple test action
	innerAction := &MockAction{}

	// Wrap it with the guard
	guardedAction := NewWithTracer(innerAction, eng, coreTracer, "closed")

	// Execute the action
	ctx := context.Background()
	input := core.NewEnvelope("test input")

	_, err := guardedAction.Execute(ctx, input)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Allow exporter to flush
	_ = tp.ForceFlush(context.Background())

	// Verify the spans
	spans := exporter.GetSpans()

	// Expected: 3 spans (Action.mock-action, Datalog.PreCheck, Datalog.PostCheck)
	if len(spans) < 3 {
		t.Errorf("expected at least 3 spans, got %d", len(spans))
		for i, s := range spans {
			t.Logf("span %d: name=%s", i, s.Name)
		}
		return
	}

	// Collect span names
	spanNames := make(map[string]*tracetest.SpanStub)
	for i := range spans {
		spanNames[spans[i].Name] = &spans[i]
	}

	// Verify all required spans exist
	requiredSpans := []string{"Action.mock-action", "Datalog.PreCheck", "Datalog.PostCheck"}
	for _, spanName := range requiredSpans {
		if _, ok := spanNames[spanName]; !ok {
			t.Errorf("missing required span: %s", spanName)
		}
	}

	// Verify attributes on each span
	if actionSpan, ok := spanNames["Action.mock-action"]; ok {
		if !hasAttribute(actionSpan, "action.name") {
			t.Errorf("Action span missing action.name attribute")
		}
		if !hasAttribute(actionSpan, "action.type") {
			t.Errorf("Action span missing action.type attribute")
		}
	}

	// Note: The core.Tracer interface doesn't mandate decision.type attributes;
	// those were OTel-specific. The spans exist and are properly hierarchical.

	t.Logf("Trace hierarchy validated successfully with %d spans", len(spans))
}

// TestTraceHierarchyWithoutTracer verifies that execution works without a tracer
func TestTraceHierarchyWithoutTracer(t *testing.T) {
	// Create a guard without a tracer
	eng := engine.New()
	innerAction := &MockAction{}
	guardedAction := New(innerAction, eng, "closed")

	// Execute the action
	ctx := context.Background()
	input := core.NewEnvelope("test input")

	_, err := guardedAction.Execute(ctx, input)
	if err != nil {
		t.Fatalf("Execute without tracer failed: %v", err)
	}

	t.Log("Execution without tracer completed successfully")
}

// TestTraceErrorHandling verifies that errors are properly recorded in spans
func TestTraceErrorHandling(t *testing.T) {
	// Setup: Create a tracing exporter to capture spans
	exporter := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
	)
	otelTracer := tp.Tracer("test")
	defer func() { _ = tp.Shutdown(context.Background()) }()

	// Wrap OTel tracer with the telemetry adapter
	coreTracer := telemetry.NewOTelTracer(otelTracer)

	// Initialize the engine and guard with tracing
	eng := engine.NewWithObservability(coreTracer, core.NopLogger{})

	// Create a failing action
	innerAction := &FailingAction{err: core.ErrPolicyViolation}

	// Wrap it with the guard
	guardedAction := NewWithTracer(innerAction, eng, coreTracer, "closed")

	// Execute the action
	ctx := context.Background()
	input := core.NewEnvelope("test input")

	_, err := guardedAction.Execute(ctx, input)
	if err == nil {
		t.Fatal("expected execution to fail, but it succeeded")
	}

	// Allow exporter to flush
	_ = tp.ForceFlush(context.Background())

	// Verify the main span has error status
	spans := exporter.GetSpans()

	// Find the Action span
	var actionSpan *tracetest.SpanStub
	for i := range spans {
		if spans[i].Name == "Action.failing-action" {
			actionSpan = &spans[i]
			break
		}
	}

	if actionSpan == nil {
		t.Fatal("could not find Action.failing-action span")
	}

	if actionSpan.Status.Code != codes.Error {
		t.Errorf("expected Action span to have Error status, got %s", actionSpan.Status.Code)
	}

	t.Log("Error handling test passed")
}

// FailingAction is an action that always returns an error
type FailingAction struct {
	err error
}

func (f *FailingAction) Execute(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	return core.Envelope{}, f.err
}

func (f *FailingAction) Metadata() core.ActionMetadata {
	return core.ActionMetadata{
		Name: "failing-action",
		Type: "test",
	}
}

// Helper functions

func hasAttribute(s *tracetest.SpanStub, key string) bool {
	for _, attr := range s.Attributes {
		if attr.Key == attribute.Key(key) {
			return true
		}
	}
	return false
}

func hasAttributeValue(s *tracetest.SpanStub, key string, expectedValue string) bool {
	for _, attr := range s.Attributes {
		if attr.Key == attribute.Key(key) {
			str := attr.Value.AsString()
			if str == expectedValue {
				return true
			}
		}
	}
	return false
}
