package supervisor

import (
	"context"
	"fmt"
	"testing"

	"github.com/duynguyendang/manglekit/core"
)

// mockExtractorForTest implements Extractor for testing.
type mockExtractorForTest struct {
	called bool
	input  string
	result any
	err    error
}

func (m *mockExtractorForTest) Extract(ctx context.Context, text string) (any, error) {
	m.called = true
	m.input = text
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

// mockSDKAction captures the payload for inspection.
type mockSDKAction struct {
	capturedPayload any
}

func (m *mockSDKAction) Execute(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	m.capturedPayload = input.Payload
	return input, nil
}

func (m *mockSDKAction) Metadata() core.ActionMetadata {
	return core.ActionMetadata{Name: "mock_sdk_action"}
}

// passThroughEvaluator is a minimal core.Evaluator that always passes.
type passThroughEvaluator struct{}

func (p *passThroughEvaluator) AssessPlan(ctx context.Context, input core.Envelope) (core.Decision, error) {
	return core.Decision{Outcome: core.DecisionProceed}, nil
}
func (p *passThroughEvaluator) Assess(ctx context.Context, meta core.ActionMetadata, input core.Envelope) error {
	return nil
}
func (p *passThroughEvaluator) Reflect(ctx context.Context, meta core.ActionMetadata, output core.Envelope) (core.Envelope, error) {
	return output, nil
}
func (p *passThroughEvaluator) EvaluateSteering(ctx context.Context, input core.Envelope) (string, map[string]string, error) {
	return core.DecisionProceed, nil, nil
}
func (p *passThroughEvaluator) GetActionConfig(ctx context.Context, input core.Envelope) (map[string]string, error) {
	return nil, nil
}
func (p *passThroughEvaluator) CheckRequirement(ctx context.Context, input core.Envelope, reqName string) (bool, error) {
	return false, nil
}
func (p *passThroughEvaluator) LoadPolicy(ctx context.Context, source string) error { return nil }
func (p *passThroughEvaluator) LoadFromSource(ctx context.Context, source string) error {
	return nil
}
func (p *passThroughEvaluator) RegisterExternalPredicate(name string, fn func(ctx context.Context, inputs []any) ([][]any, error)) error {
	return nil
}
func (p *passThroughEvaluator) LoadGherkinPolicy(ctx context.Context, f string) error { return nil }
func (p *passThroughEvaluator) LoadFacts(ctx context.Context, facts []string) error   { return nil }
func (p *passThroughEvaluator) RegisterAction(meta core.ActionMetadata) error         { return nil }
func (p *passThroughEvaluator) Query(ctx context.Context, facts []string, query string) ([]map[string]string, error) {
	return nil, nil
}
func (p *passThroughEvaluator) Logger() core.Logger { return core.NopLogger{} }

// makeTestSV2 creates a supervisedActionV2 using the production constructor.
func makeTestSV2(inner core.Action) *supervisedActionV2 {
	action := NewSupervisedActionFromSDK(inner, &passThroughEvaluator{}, core.NopLogger{})
	return action.(*supervisedActionV2)
}

// TestSupervisedActionV2_TextExtraction verifies extractor is called for text.
func TestSupervisedActionV2_TextExtraction(t *testing.T) {
	type ExtractedData struct {
		Amount float64 `mangle:"amount"`
		Status string  `mangle:"status"`
	}

	inner := &mockSDKAction{}
	sv := makeTestSV2(inner)

	ext := &mockExtractorForTest{
		result: ExtractedData{Amount: 100.0, Status: "completed"},
	}
	sv.extractor = ext

	input := core.NewEnvelope("I'll transfer $100 to acct 9927")
	_, err := sv.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	if !ext.called {
		t.Fatal("extractor should have been called for text payload")
	}
	if ext.input != "I'll transfer $100 to acct 9927" {
		t.Errorf("extractor got wrong input: %q", ext.input)
	}

	if inner.capturedPayload == nil {
		t.Fatal("inner action was not called")
	}
	payload, ok := inner.capturedPayload.(ExtractedData)
	if !ok {
		t.Fatalf("expected ExtractedData, got %T: %v", inner.capturedPayload, inner.capturedPayload)
	}
	if payload.Amount != 100.0 {
		t.Errorf("expected amount 100.0, got %v", payload.Amount)
	}
}

// TestSupervisedActionV2_NoExtractor verifies no-op when extractor is nil.
func TestSupervisedActionV2_NoExtractor(t *testing.T) {
	inner := &mockSDKAction{}
	sv := makeTestSV2(inner)

	input := core.NewEnvelope("free text payload")
	_, err := sv.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	text, ok := inner.capturedPayload.(string)
	if !ok {
		t.Fatalf("expected string, got %T", inner.capturedPayload)
	}
	if text != "free text payload" {
		t.Errorf("expected 'free text payload', got %q", text)
	}
}

// TestSupervisedActionV2_ExtractorError verifies error propagation.
func TestSupervisedActionV2_ExtractorError(t *testing.T) {
	inner := &mockSDKAction{}
	sv := makeTestSV2(inner)

	sv.extractor = &mockExtractorForTest{
		err: fmt.Errorf("extraction failed: schema mismatch"),
	}

	input := core.NewEnvelope("some text")
	_, err := sv.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error from extraction failure")
	}
}

// TestSupervisedActionV2_StructPayloadSkipsExtractor verifies structs bypass extractor.
func TestSupervisedActionV2_StructPayloadSkipsExtractor(t *testing.T) {
	inner := &mockSDKAction{}
	sv := makeTestSV2(inner)

	ext := &mockExtractorForTest{result: "should not be reached"}
	sv.extractor = ext

	type TestData struct {
		Name string `mangle:"name"`
	}

	input := core.NewEnvelope(TestData{Name: "test"})
	_, err := sv.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	if ext.called {
		t.Error("extractor should not be called for struct payloads")
	}
}

func TestWithExtractor(t *testing.T) {
	inner := &mockSDKAction{}
	sv := makeTestSV2(inner)

	ext := &mockExtractorForTest{result: "extracted"}
	action := WithExtractor(sv, ext)

	svResult, ok := action.(*supervisedActionV2)
	if !ok {
		t.Fatal("WithExtractor should return *supervisedActionV2")
	}
	if svResult.extractor != ext {
		t.Error("extractor was not set")
	}
}
